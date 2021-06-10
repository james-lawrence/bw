package interp

import (
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func exportEnvx() (exported map[string]reflect.Value) {
	exported = map[string]reflect.Value{
		// Boolean retrieve a boolean flag from the environment, checks each key in order
		// first to parse successfully is returned.
		"Boolean": reflect.ValueOf(func(fallback bool, values ...string) bool {
			for _, k := range values {
				if b, err := strconv.ParseBool(k); err == nil {
					return b
				}
			}

			return fallback
		}),
		// String retrieve a string value from the environment, checks each key in order
		// first string found is returned.
		"String": reflect.ValueOf(func(fallback string, values ...string) string {
			for _, k := range values {
				s := strings.TrimSpace(k)
				if s != "" {
					return s
				}
			}

			return fallback
		}),

		// Duration retrieves a time.Duration from the environment, checks each key in order
		// first successful parse to a duration is returned.
		"Duration": reflect.ValueOf(func(fallback time.Duration, v ...string) time.Duration {
			for _, k := range v {
				s := strings.TrimSpace(k)
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
		}),
	}

	return exported
}
