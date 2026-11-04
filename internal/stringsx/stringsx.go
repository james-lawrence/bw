package stringsx

import "strings"

// Join a more convient string join method.
func Join(sep string, parts ...string) string {
	return strings.Join(parts, sep)
}

// Reverse returns the string reversed rune-wise left to right.
func Reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func Empty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// First get the first non blank from the array.
func First(values ...string) string {
	for _, s := range values {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}

	return ""
}
