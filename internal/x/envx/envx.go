// Package envx provides utility functions for extracting information from environment variables
package envx

import (
	"os"
	"strconv"
)

// Boolean retrieve a boolean flag from the environment, checks each variable in order
// first to parse successfully is returned.
func Boolean(fallback bool, keys ...string) bool {
	for _, k := range keys {
		if b, err := strconv.ParseBool(os.Getenv(k)); err == nil {
			return b
		}
	}

	return fallback
}
