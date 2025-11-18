package util

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"golang.org/x/exp/slices"
)

var HeaderBlacklist = []string{"Cookie", "X-CSRFToken"}

// These headers aren't things that proxies should pass along. Some are forbidden by http2.
// This fixes the bug where Chrome users saw a ERR_SPDY_PROTOCOL_ERROR for all proxied requests.
func FilterHeaders(r *http.Response) error {
	badHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Connection",
		"Transfer-Encoding",
		"Upgrade",
		"Access-Control-Allow-Headers",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Origin",
		"Access-Control-Expose-Headers",
	}
	for _, h := range badHeaders {
		r.Header.Del(h)
	}
	return nil
}

func SingleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// decodeSubprotocol decodes the impersonation "headers" on a websocket.
// Subprotocols don't allow '=' or '/'
func DecodeSubprotocol(encodedProtocol string) (string, error) {
	encodedProtocol = strings.Replace(encodedProtocol, "_", "=", -1)
	encodedProtocol = strings.Replace(encodedProtocol, "-", "/", -1)
	decodedProtocol, err := base64.StdEncoding.DecodeString(encodedProtocol)
	return string(decodedProtocol), err
}

func CopyMsgs(writeMutex *sync.Mutex, dest *websocket.Conn, src *websocket.Conn) error {
	for {
		messageType, msg, err := src.ReadMessage()
		if err != nil {
			return err
		}

		if writeMutex == nil {
			err = dest.WriteMessage(messageType, msg)
		} else {
			writeMutex.Lock()
			err = dest.WriteMessage(messageType, msg)
			writeMutex.Unlock()
		}

		if err != nil {
			return err
		}
	}
}

func KeepAlive(writeMutex *sync.Mutex, dest *websocket.Conn) error {
	websocketTimeout := 30 * time.Second
	websocketPingInterval := 30 * time.Second
	ticker := time.NewTicker(websocketPingInterval)

	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			writeMutex.Lock()
			// Send pings to client to prevent load balancers and other middlemen from closing the connection early
			err := dest.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(websocketTimeout))
			writeMutex.Unlock()
			if err != nil {
				return err
			}
		}
	}
}

func CreateProxyHeaders(w http.ResponseWriter, r *http.Request) (http.Header, string, error) {
	subProtocol := ""
	proxiedHeader := make(http.Header, len(r.Header))
	for key, value := range r.Header {
		if key != "Sec-Websocket-Protocol" {
			// Do not proxy the subprotocol to the API server because k8s does not understand what we're sending
			proxiedHeader.Set(key, r.Header.Get(key))
			continue
		}

		for _, protocols := range value {
			for _, protocol := range strings.Split(protocols, ",") {
				protocol = strings.TrimSpace(protocol)
				// TODO: secure by stripping newlines & other invalid stuff
				// "Impersonate-User" and "Impersonate-Group" and bridge specific (not a k8s thing)
				if strings.HasPrefix(protocol, "Impersonate-User.") {
					encodedProtocol := strings.TrimPrefix(protocol, "Impersonate-User.")
					decodedProtocol, err := DecodeSubprotocol(encodedProtocol)
					if err != nil {
						errMsg := fmt.Sprintf("Error decoding Impersonate-User subprotocol: %v", err)
						http.Error(w, errMsg, http.StatusBadRequest)
						return nil, "", err
					}
					proxiedHeader.Set("Impersonate-User", decodedProtocol)
					subProtocol = protocol
				} else if strings.HasPrefix(protocol, "Impersonate-Group.") {
					encodedProtocol := strings.TrimPrefix(protocol, "Impersonate-Group.")
					decodedProtocol, err := DecodeSubprotocol(encodedProtocol)
					if err != nil {
						errMsg := fmt.Sprintf("Error decoding Impersonate-Group subprotocol: %v", err)
						http.Error(w, errMsg, http.StatusBadRequest)
						return nil, "", err
					}
					proxiedHeader.Set("Impersonate-User", string(decodedProtocol))
					proxiedHeader.Set("Impersonate-Group", string(decodedProtocol))
					subProtocol = protocol
				} else {
					proxiedHeader.Set("Sec-Websocket-Protocol", protocol)
					subProtocol = protocol
				}
			}
		}
	}

	// Filter websocket headers.
	websocketHeaders := []string{
		"Connection",
		"Sec-Websocket-Extensions",
		"Sec-Websocket-Key",
		// NOTE: kans - Sec-Websocket-Protocol must be proxied in the headers
		"Sec-Websocket-Version",
		"Upgrade",
	}
	for _, header := range websocketHeaders {
		proxiedHeader.Del(header)
	}

	return proxiedHeader, subProtocol, nil
}

func labelsIncludes(labels map[string]interface{}, label string) bool {
	splitLabel := strings.Split(label, "=")
	return labels[splitLabel[0]] == splitLabel[1]
}

func isMigratable(statuses []interface{}, search string) bool {
	indexOfItem := slices.IndexFunc(statuses, func(status interface{}) bool {
		return status.(map[string]interface{})["type"] == "LiveMigratable" && status.(map[string]interface{})["status"] == "True"
	})
	isMigrate := indexOfItem != -1
	if search == "notMigratable" && isMigrate {
		return false
	}

	if search == "migratable" && !isMigrate {
		return false
	}

	return true
}

func isIPExist(interfaces []interface{}, search string) bool {
	result := false
	for _, nic := range interfaces {
		nicsArr := nic.(map[string]interface{})["ipAddresses"].([]interface{})
		isExist := slices.ContainsFunc(nicsArr, func(ip interface{}) bool {
			return strings.Contains(ip.(string), search)
		})
		if isExist {
			result = true
			break
		}
	}
	return result
}

func itemValueMatchesFilter(itemValue gjson.Result, key string, queryValue string) bool {
	searchValues := strings.Split(queryValue, ",")

	switch itemValueType := itemValue.Type.String(); itemValueType {
	case "JSON":
		// ALL search values must match (AND logic)
		for _, search := range searchValues {
			switch key {
			case "status.conditions":
				if !isMigratable(itemValue.Value().([]interface{}), search) {
					return false
				}
			case "status.interfaces":
				if !isIPExist(itemValue.Value().([]interface{}), search) {
					return false
				}
			default:
				if !labelsIncludes(itemValue.Value().(map[string]interface{}), search) {
					return false
				}
			}
		}
		return true

	case "String":
		// ANY search value can match (OR logic)
		for _, search := range searchValues {
			if strings.Contains(strings.ToLower(itemValue.Str), strings.ToLower(search)) {
				return true
			}
		}
		return false

	case "Null":
		// ALL search values must be "null" (AND logic)
		for _, search := range searchValues {
			if strings.ToLower(search) != "null" {
				return false
			}
		}
		return true

	default:
		return false // Unsupported type
	}
}

const KEY_DELIMITER = "|"

func FilterResponseQuery(bodyBytes []byte, query url.Values) map[string]interface{} {
	items := gjson.ParseBytes(bodyBytes).Get("items").Array()
	filteredJson := []interface{}{}
	isFilters := len(query) != 0
	if isFilters {
	nextItem:
		for _, item := range items {
			for key, values := range query {
				// values is a slice, because same key can repeat in the query string
				// e.g. same.key=value1,value2&same.key=value3 results in values = []string{"value1,value2", "value3"}
				// in Kubevirt UI the query string is created such way that the key doesn't repeat, so we can use values[0]
				value := values[0]

				if strings.Contains(key, KEY_DELIMITER) {
					keys := strings.Split(key, KEY_DELIMITER)
					keyMatched := false

					for _, key := range keys {
						itemValue := item.Get(key)
						if itemValueMatchesFilter(itemValue, key, value) {
							keyMatched = true
							break
						}
					}

					if !keyMatched {
						continue nextItem
					}
				} else {
					itemValue := item.Get(key)
					if !itemValueMatchesFilter(itemValue, key, value) {
						continue nextItem
					}
				}
			}

			valueJson := map[string]interface{}{}
			err := json.Unmarshal([]byte(item.Raw), &valueJson)
			if err != nil {
				log.Println("error creating json of item: ", err.Error())
			} else {
				filteredJson = append(filteredJson, valueJson)
			}
		}
	}

	returnJson := map[string]interface{}{}
	json.Unmarshal(bodyBytes, &returnJson)
	returnJson["totalItems"] = len(items)
	if isFilters {
		returnJson["items"] = filteredJson
	}

	return returnJson
}
