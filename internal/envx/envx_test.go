package envx_test

import (
	"testing"

	"github.com/james-lawrence/bw/internal/envx"
	"github.com/stretchr/testify/require"
)

func TestIsolate(t *testing.T) {
	environ := []string{
		"PATH=/usr/bin",
		"BEARDED_WOOKIE_AGENT_P2P_BIND=127.0.0.1:2000",
		"HOME=/home/egd",
		"BEARDED_WOOKIE_LOGS_RAFT=true",
		"EMPTY=",
		"malformed",
	}

	for name, tc := range map[string]struct {
		globs    []string
		expected []string
	}{
		"no globs keeps everything": {
			expected: environ,
		},
		"prefix glob": {
			globs:    []string{"BEARDED_WOOKIE_*"},
			expected: []string{"PATH=/usr/bin", "HOME=/home/egd", "EMPTY=", "malformed"},
		},
		"exact name": {
			globs:    []string{"HOME"},
			expected: []string{"PATH=/usr/bin", "BEARDED_WOOKIE_AGENT_P2P_BIND=127.0.0.1:2000", "BEARDED_WOOKIE_LOGS_RAFT=true", "EMPTY=", "malformed"},
		},
		"multiple globs": {
			globs:    []string{"BEARDED_WOOKIE_LOGS_*", "HOME", "EMPTY"},
			expected: []string{"PATH=/usr/bin", "BEARDED_WOOKIE_AGENT_P2P_BIND=127.0.0.1:2000", "malformed"},
		},
		"matches names not values": {
			globs:    []string{"*127*"},
			expected: environ,
		},
		"malformed entries are never dropped": {
			globs:    []string{"malformed", "*"},
			expected: []string{"malformed"},
		},
		"an invalid pattern only matches the exact name": {
			globs:    []string{"[", "PATH"},
			expected: []string{"BEARDED_WOOKIE_AGENT_P2P_BIND=127.0.0.1:2000", "HOME=/home/egd", "BEARDED_WOOKIE_LOGS_RAFT=true", "EMPTY=", "malformed"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.expected, envx.Isolate(environ, tc.globs...))
		})
	}

	t.Run("does not modify its input", func(t *testing.T) {
		in := []string{"A=1", "B=2"}
		_ = envx.Isolate(in, "A")
		require.Equal(t, []string{"A=1", "B=2"}, in)
	})
}
