package stringsx

import "strings"

// Default returns the default value if val is empty or composed entirely of spaces.
func Default(val, def string) string {
	if strings.TrimSpace(val) == "" {
		return def
	}

	return val
}
