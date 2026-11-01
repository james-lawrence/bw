package errorsx_test

import (
	"errors"
	"testing"

	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/stretchr/testify/require"
)

func TestFrom(t *testing.T) {
	boom := errors.New("boom")
	require.NoError(t, errorsx.From(1, nil))
	require.ErrorIs(t, errorsx.From(1, boom), boom)
}
