package proxy_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/james-lawrence/bw/proxy"
	"github.com/stretchr/testify/require"
)

func TestWireformatRoundTrip(t *testing.T) {
	for _, in := range [][]byte{{}, []byte("hello"), bytes.Repeat([]byte{7}, proxy.MaxWireformat)} {
		out, err := proxy.WireformatDecode(bytes.NewReader(proxy.WireformatEncode(in)))
		require.NoError(t, err)
		require.Equal(t, len(in), len(out))
		require.True(t, bytes.Equal(in, out))
	}
}

// the length prefix is unauthenticated, an oversized one is rejected without allocating or reading.
func TestWireformatRejectsOversizedLength(t *testing.T) {
	for _, n := range []uint64{proxy.MaxWireformat + 1, 1 << 40, ^uint64(0)} {
		hdr := make([]byte, 8)
		binary.LittleEndian.PutUint64(hdr, n)
		out, err := proxy.WireformatDecode(bytes.NewReader(hdr))
		require.ErrorIs(t, err, proxy.ErrWireformatTooLarge, "length(%d)", n)
		require.Nil(t, out)
	}
}

func TestWireformatTruncated(t *testing.T) {
	encoded := proxy.WireformatEncode([]byte("hello"))
	_, err := proxy.WireformatDecode(bytes.NewReader(encoded[:len(encoded)-1]))
	require.Error(t, err)
	_, err = proxy.WireformatDecode(bytes.NewReader(encoded[:3]))
	require.Error(t, err)
}
