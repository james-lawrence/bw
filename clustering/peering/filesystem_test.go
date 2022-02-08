package peering_test

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/clusteringtestutil"

	. "github.com/james-lawrence/bw/clustering/peering"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("File", func() {
	var (
		err    error
		tmpdir string
	)

	BeforeEach(func() {
		tmpdir, err = ioutil.TempDir(".", "fs-peering")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpdir)
	})

	It("should persist the peers to a file", func() {
		c := clusteringtestutil.NewMock(3)
		fs := File{Path: filepath.Join(tmpdir, "peers.yml")}
		Expect(fs.Snapshot(clustering.Peers(c))).ToNot(HaveOccurred())

		peers, err := fs.Peers(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(peers).To(ConsistOf(clustering.Peers(c)))
	})
})
