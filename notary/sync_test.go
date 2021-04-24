package notary_test

import (
	"context"
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/willf/bloom"
	"google.golang.org/grpc"

	"github.com/james-lawrence/bw/internal/x/iox"
	"github.com/james-lawrence/bw/internal/x/testingx"
	"github.com/james-lawrence/bw/notary"
)

func synced(stream notary.Sync_StreamClient, s storage) (err error) {
	for {
		var (
			event *notary.SyncStream
		)

		if event, err = stream.Recv(); err != nil {
			err = iox.IgnoreEOF(err)
			break
		}

		switch evt := event.Events.(type) {
		case *notary.SyncStream_Chunk:
			for _, g := range evt.Chunk.Grants {
				log.Println("retrieved", g.Fingerprint)
				if _, err := s.Insert(g); err != nil {
					return err
				}
			}
		}
	}

	return err
}

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
		Expect(synced(stream, s2)).To(Succeed())
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
