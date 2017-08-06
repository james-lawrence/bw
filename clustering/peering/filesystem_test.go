package peering_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/clustering/clusteringtestutil"

	. "bitbucket.org/jatone/bearded-wookie/clustering/peering"

	. "github.com/onsi/ginkgo"
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

		peers, err := fs.Peers()
		Expect(err).ToNot(HaveOccurred())
		Expect(peers).To(ConsistOf(clustering.Peers(c)))
	})
})
