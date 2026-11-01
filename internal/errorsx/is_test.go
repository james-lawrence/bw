package errorsx_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/stretchr/testify/require"
)

func TestIs(t *testing.T) {
	boom := errors.New("boom")
	wrapped := fmt.Errorf("wrapped: %w", boom)

	t.Run("matches any of the protos", func(t *testing.T) {
		require.True(t, errorsx.Is(wrapped, context.Canceled, boom))
	})

	t.Run("does not match when none of the protos match", func(t *testing.T) {
		require.False(t, errorsx.Is(wrapped, context.Canceled))
	})

	t.Run("does not match without protos", func(t *testing.T) {
		require.False(t, errorsx.Is(wrapped))
	})

	t.Run("does not match a nil cause", func(t *testing.T) {
		require.False(t, errorsx.Is(nil, boom))
	})
}
