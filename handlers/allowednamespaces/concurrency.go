package allowednamespaces

import (
	"fmt"
	"os"
	"strconv"
)

const (
	maxConcurrencyEnvVar    = "ALLOWED_NS_MAX_CONCURRENCY"
	maxConcurrencyCapEnvVar = "ALLOWED_NS_MAX_CONCURRENCY_CAP"

	fallbackMaxConcurrentSSARChecks = 20
	fallbackMaxConcurrentSSARCap    = 100
)

var (
	defaultMaxConcurrentSSARChecks = envInt(maxConcurrencyEnvVar, fallbackMaxConcurrentSSARChecks)
	maxConcurrentSSARChecksCap       = envInt(maxConcurrencyCapEnvVar, fallbackMaxConcurrentSSARCap)
)

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}

	return parsed
}

func resolveMaxConcurrency(requestValue *int) (int, error) {
	if requestValue == nil {
		return defaultMaxConcurrentSSARChecks, nil
	}

	maxConcurrency := *requestValue
	if maxConcurrency < 1 {
		return 0, fmt.Errorf("maxConcurrency must be a positive integer")
	}

	if maxConcurrency > maxConcurrentSSARChecksCap {
		maxConcurrency = maxConcurrentSSARChecksCap
	}

	return maxConcurrency, nil
}
