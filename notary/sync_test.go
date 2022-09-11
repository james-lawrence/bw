package notary_test

import (
	"context"

	"github.com/bits-and-blooms/bloom/v3"
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
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

var _ = Describe("SyncServer", func() {
	var (
		g1 = QuickGrant()
		g2 = QuickGrant()
		g3 = QuickGrant()
		g4 = QuickGrant()
	)

	DescribeTable("return missing grants", func(b notary.Bloomy, s1 notary.SyncStorage, s2 storage, expected ...*notary.Grant) {
		d, srv := testingx.NewGRPCServer2(func(s *grpc.Server) {
			notary.NewSyncService(staticauth{Permission: all()}, s1).Bind(s)
		})
		defer testingx.GRPCCleanup(nil, srv)

		conn, err := d.Dial()
		Expect(err).To(Succeed())

		client := notary.NewSyncClient(conn)

		req, err := notary.NewSyncRequest(bloom.NewWithEstimates(300, 0.001))
		Expect(err).To(Succeed())

		stream, err := client.Stream(context.Background(), req)
		Expect(err).To(Succeed())
		Expect(notary.Sync(stream, b, s2)).To(Succeed())
		for _, g := range expected {
			_, err := s2.Lookup(g.Fingerprint)
			Expect(err).To(Succeed())
		}
	},
		Entry(
			"given an empty bloom, return all grants",
			bloom.NewWithEstimates(300, 0.001),
			notary.NewMem(g1, g2, g3, g4),
			notary.NewMem(),
			g1, g2, g3, g4,
		),
		Entry(
			"given a partial bloom, return missing grants",
			newBloom(bloom.NewWithEstimates(300, 0.001), g2, g4),
			notary.NewMem(g1, g2, g3, g4),
			notary.NewMem(g2, g4),
			g1, g2, g3, g4,
		),
	)
})
