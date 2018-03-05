package raftutil_test

import (
	"io/ioutil"
	"log"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRaftutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Raftutil Suite")
}

var (
	tmpdir string
)
var _ = BeforeSuite(func() {
	var (
		err error
	)
	log.SetOutput(ioutil.Discard)
	tmpdir, err = ioutil.TempDir(".", "sockets")
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	defer os.RemoveAll(tmpdir)
})
