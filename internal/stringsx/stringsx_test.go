package stringsx_test

import (
	"testing"

	"github.com/james-lawrence/bw/internal/stringsx"
	"github.com/stretchr/testify/require"
)

func TestTail(t *testing.T) {
	require.Equal(t, "cdef", stringsx.Tail("abcdef", 4))
	require.Equal(t, "abc", stringsx.Tail("abc", 8))
	require.Equal(t, "", stringsx.Tail("abc", 0))
	require.Equal(t, "世界", stringsx.Tail("你好世界", 2))
}
