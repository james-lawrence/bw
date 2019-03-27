package raftutil_test

import (
	"io/ioutil"
	"log"

	"github.com/james-lawrence/bw/internal/x/testingx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRaftutil(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Raftutil Suite")
}

var _ = SynchronizedAfterSuite(func() {}, func() {
	testingx.Cleanup()
})
