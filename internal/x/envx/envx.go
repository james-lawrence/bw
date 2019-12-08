// Package envx provides utility functions for extracting information from environment variables
package envx

import (
	"os"
	"strconv"
	"strings"
)

// Boolean retrieve a boolean flag from the environment, checks each key in order
// first to parse successfully is returned.
func Boolean(fallback bool, keys ...string) bool {
	for _, k := range keys {
		if b, err := strconv.ParseBool(os.Getenv(k)); err == nil {
			return b
		}
	}

	return fallback
}

// String retrieve a string value from the environment, checks each key in order
// first string found is returned.
func String(fallback string, keys ...string) string {
	for _, k := range keys {
		s := strings.TrimSpace(os.Getenv(k))
		if s != "" {
			return s
		}
	}

	return fallback
}
