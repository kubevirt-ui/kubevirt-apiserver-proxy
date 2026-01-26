package config

import (
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
}

var (
	minTLSVersionFlag   = flag.Uint("tls-min-version", 0, "The minimum TLS version to use")
	tlsCipherSuitesFlag = flag.String("tls-cipher-suites", "", "A comma-separated list of cipher suites to use")
)

func GetConfig() (*Config, error) {
	cfg := &Config{}

	if *minTLSVersionFlag > 0 {
		if *minTLSVersionFlag > math.MaxUint16 {
			return nil, fmt.Errorf("the --tls-min-version flag is with a wrong value:  %d is lager than the max allowed value of %d", *minTLSVersionFlag, math.MaxUint16)
		}

		cfg.TLS.minTLSVersion = uint16(*minTLSVersionFlag)
	}

	if *tlsCipherSuitesFlag != "" {
		ciphers := strings.Split(*tlsCipherSuitesFlag, ",")
		tlsCipherSuites := make([]uint16, 0, len(ciphers))

		for _, cipherStr := range ciphers {
			cipherStr = strings.TrimSpace(cipherStr)
			cipher, err := strconv.ParseUint(cipherStr, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("can't parse cipher %q; %w", cipherStr, err)
			}

			tlsCipherSuites = append(tlsCipherSuites, uint16(cipher))
		}

		cfg.TLS.tlsCipherSuites = tlsCipherSuites
	}

	return cfg, nil
}

func (cfg *Config) GetMinTLSVersion() uint16 {
	return cfg.TLS.minTLSVersion
}

func (cfg *Config) GetTLSCipherSuites() []uint16 {
	return cfg.TLS.tlsCipherSuites
}
