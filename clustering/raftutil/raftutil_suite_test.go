package raftutil_test

import (
	"log"

	"github.com/james-lawrence/bw/internal/x/testingx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRaftutil(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	// log.SetOutput(ioutil.Discard)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Raftutil Suite")
}

var _ = SynchronizedAfterSuite(func() {}, testingx.Cleanup)
