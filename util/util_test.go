package util

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/tidwall/gjson"
)

// gjsonName extracts metadata.name from a filtered item (which FilterResponseQuery returns as a
// decoded map[string]interface{}).
func gjsonName(t *testing.T, item interface{}) string {
	t.Helper()
	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal item: %v", err)
	}
	return gjson.GetBytes(raw, "metadata.name").String()
}

func TestResourceFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{name: "namespaced list", path: "/apis/kubevirt.io/v1/namespaces/default/virtualmachines", expected: "virtualmachines"},
		{name: "namespaced get", path: "/apis/kubevirt.io/v1/namespaces/default/virtualmachines/my-vm", expected: "virtualmachines"},
		{name: "cluster-scoped list", path: "/apis/kubevirt.io/v1/virtualmachineinstances", expected: "virtualmachineinstances"},
		{name: "with proxy prefix before apis", path: "/some/proxy/apis/kubevirt.io/v1/namespaces/ns/templates", expected: "templates"},
		{name: "no apis segment", path: "/healthz", expected: ""},
		{name: "too short", path: "/apis/kubevirt.io", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResourceFromPath(tt.path); got != tt.expected {
				t.Errorf("ResourceFromPath(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

// TestFilterResponseQuery_NamedOS verifies the named "os" key is routed through the virtualmachines
// matcher end-to-end (and is not treated as a yaml path) when the resource is virtualmachines.
func TestFilterResponseQuery_NamedOS(t *testing.T) {
	body := []byte(`{
		"apiVersion": "kubevirt.io/v1",
		"kind": "VirtualMachineList",
		"items": [
			{"metadata":{"name":"vm-rhel"},"spec":{"preference":{"name":"rhel.9"}}},
			{"metadata":{"name":"vm-fedora"},"spec":{"preference":{"name":"fedora"}}}
		]
	}`)

	result := FilterResponseQuery(body, url.Values{"os": {"RHEL"}}, "virtualmachines")

	if total, ok := result["totalItems"].(int); !ok || total != 2 {
		t.Fatalf("totalItems = %v, want 2", result["totalItems"])
	}

	items, ok := result["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("filtered items = %v, want exactly 1", result["items"])
	}

	if name := gjsonName(t, items[0]); name != "vm-rhel" {
		t.Errorf("filtered item = %q, want %q", name, "vm-rhel")
	}
}
