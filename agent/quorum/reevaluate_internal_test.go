package quorum

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestReevaluates(t *testing.T) {
	for name, tc := range map[string]struct {
		err      error
		expected bool
	}{
		"state machine rejection": {status.Error(codes.FailedPrecondition, "deploy already in progress"), false},
		"duplicate message":       {status.Error(codes.AlreadyExists, "duplicate"), false},
		"overloaded leader":       {status.Error(codes.ResourceExhausted, "back off"), false},
		"caller gave up":          {status.Error(codes.Canceled, ""), false},
		"deadline":                {status.Error(codes.DeadlineExceeded, ""), false},
		"leader unavailable":      {status.Error(codes.Unavailable, "not leader"), true},
		"unclassified":            {errors.New("boom"), true},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.expected, reevaluates(tc.err))
		})
	}
}
