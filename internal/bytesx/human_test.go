package bytesx_test

import (
	"testing"

	"github.com/james-lawrence/bw/internal/bytesx"
)

func TestHuman(t *testing.T) {
	for _, tc := range []struct {
		in       uint64
		expected string
	}{
		{0, "0.0 B"},
		{512, "512.0 B"},
		{bytesx.MiB, "1.0 MiB"},
		{bytesx.MiB + bytesx.MiB/2, "1.5 MiB"},
		{2 * bytesx.GiB, "2.0 GiB"},
	} {
		if actual := bytesx.Human(tc.in); actual != tc.expected {
			t.Errorf("Human(%d) = %q, expected %q", tc.in, actual, tc.expected)
		}
	}
}
