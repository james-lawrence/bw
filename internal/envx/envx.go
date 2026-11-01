// Package envx provides utility functions for extracting information from environment variables
package envx

import (
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/james-lawrence/bw/internal/langx"
	"github.com/pkg/errors"
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
func String[T ~string | ~rune](fallback T, keys ...string) string {
	for _, k := range keys {
		s := strings.TrimSpace(os.Getenv(k))
		if s != "" {
			return s
		}
	}

	return string(fallback)
}

// Rune retrieve a string value from the environment, checks each key in order
// first string found is returned.
func Rune[T ~rune](fallback T, keys ...string) rune {
	for _, k := range keys {
		s := strings.TrimSpace(os.Getenv(k))
		if s != "" {
			return langx.FirstNonZero([]rune(s)...)
		}
	}

	return rune(fallback)
}

// Duration retrieves a time.Duration from the environment, checks each key in order
// first successful parse to a duration is returned.
func Duration(fallback time.Duration, keys ...string) time.Duration {
	for _, k := range keys {
		s := strings.TrimSpace(os.Getenv(k))
		if s == "" {
			continue
		}

		if d, err := time.ParseDuration(s); err == nil {
			return d
		} else {
			log.Println(errors.Wrapf(err, "unable to parse time.Duration from %s", s))
		}
	}

	return fallback
}

// Isolate returns environ without the variables whose names match any of the globs, e.g.) "BEARDED_WOOKIE_*",
// the patterns are those of path.Match. A glob that is not a valid pattern matches only the exact name.
// The order of the remaining variables is preserved and entries that are not of the form key=value are kept.
func Isolate(environ []string, globs ...string) []string {
	matches := func(name string) bool {
		for _, glob := range globs {
			if ok, err := path.Match(glob, name); (err == nil && ok) || (err != nil && glob == name) {
				return true
			}
		}

		return false
	}

	isolated := make([]string, 0, len(environ))
	for _, kv := range environ {
		if name, _, ok := strings.Cut(kv, "="); ok && matches(name) {
			continue
		}

		isolated = append(isolated, kv)
	}

	return isolated
}
