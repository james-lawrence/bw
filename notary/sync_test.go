package notary_test

import (
	"context"
	"testing"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/james-lawrence/bw/internal/testingx"
	"github.com/james-lawrence/bw/notary"
)

func newBloom(b *bloom.BloomFilter, grants ...*notary.Grant) *bloom.BloomFilter {
	for _, g := range grants {
		b.Add([]byte(g.Fingerprint))
	}

	return b
}

func TestSyncServer(t *testing.T) {
	// return missing grants
	check := func(t *testing.T, b notary.Bloomy, s1 notary.SyncStorage, s2 storage, expected ...*notary.Grant) {
		d, srv := testingx.NewGRPCServer2(t, func(s *grpc.Server) {
			notary.NewSyncService(staticauth{Permission: all()}, s1).Bind(s)
		})
		defer testingx.GRPCCleanup(t, nil, srv)

		conn, err := d.DialContext(context.Background())
		require.NoError(t, err)

		client := notary.NewSyncClient(conn)

		req, err := notary.NewSyncRequest(bloom.NewWithEstimates(300, 0.001))
		require.NoError(t, err)

		stream, err := client.Stream(context.Background(), req)
		require.NoError(t, err)
		require.NoError(t, notary.Sync(stream, b, s2))
		for _, g := range expected {
			_, err := s2.Lookup(g.Fingerprint)
			require.NoError(t, err)
		}
	}

	t.Run("given an empty bloom, return all grants", func(t *testing.T) {
		g1, g2, g3, g4 := QuickGrant(t), QuickGrant(t), QuickGrant(t), QuickGrant(t)
		check(t,
			bloom.NewWithEstimates(300, 0.001),
			notary.NewMem(g1, g2, g3, g4),
			notary.NewMem(),
			g1, g2, g3, g4,
		)
	})

	t.Run("given a partial bloom, return missing grants", func(t *testing.T) {
		g1, g2, g3, g4 := QuickGrant(t), QuickGrant(t), QuickGrant(t), QuickGrant(t)
		check(t,
			newBloom(bloom.NewWithEstimates(300, 0.001), g2, g4),
			notary.NewMem(g1, g2, g3, g4),
			notary.NewMem(g2, g4),
			g1, g2, g3, g4,
		)
	})
}
