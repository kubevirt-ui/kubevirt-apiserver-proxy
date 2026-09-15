package config

import (
	"crypto/tls"
	"flag"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type Config struct {
	TLS TLSConfig
}

type TLSConfig struct {
	minTLSVersion   uint16
	tlsCipherSuites []uint16
	tlsCurves       []tls.CurveID
}

var (
	minTLSVersionFlag   = flag.Uint("tls-min-version", 0, "The minimum TLS version to use")
	tlsCipherSuitesFlag = flag.String("tls-cipher-suites", "", "A comma-separated list of cipher suites to use")
	tlsCurvesFlag       = flag.String("tls-curve-ids", "", "A comma-separated list of TLS curve IDs to use")
)

func GetConfig() (*Config, error) {
	cfg := &Config{}

	if *minTLSVersionFlag > 0 {
		if *minTLSVersionFlag > math.MaxUint16 {
			return nil, fmt.Errorf("the --tls-min-version flag is with a wrong value:  %d is lager than the max allowed value of %d", *minTLSVersionFlag, math.MaxUint16)
		}

		cfg.TLS.minTLSVersion = uint16(*minTLSVersionFlag)
	}

	ciphers, err := commaStringToList(*tlsCipherSuitesFlag, castToUint16)
	if err != nil {
		return nil, fmt.Errorf("can't parse cipher; %w", err)
	}
	cfg.TLS.tlsCipherSuites = ciphers

	curves, err := commaStringToList(*tlsCurvesFlag, castToTlsCurveId)
	if err != nil {
		return nil, fmt.Errorf("can't parse curveID; %w", err)
	}
	cfg.TLS.tlsCurves = curves

	return cfg, nil
}

func (cfg *Config) GetMinTLSVersion() uint16 {
	return cfg.TLS.minTLSVersion
}

func (cfg *Config) GetTLSCipherSuites() []uint16 {
	return cfg.TLS.tlsCipherSuites
}

func (cfg *Config) GetTLSCurveIDs() []tls.CurveID {
	return cfg.TLS.tlsCurves
}

func commaStringToList[T any](commaStr string, cast func(uint64) T) ([]T, error) {
	var result []T
	for item := range strings.SplitSeq(commaStr, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		numericItem, err := strconv.ParseUint(item, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("can't parse numeric string %q; %w", item, err)
		}
		result = append(result, cast(numericItem))
	}

	return result, nil
}

func castToUint16(val uint64) uint16 {
	return uint16(val)
}

func castToTlsCurveId(val uint64) tls.CurveID {
	return tls.CurveID(val)
}
