package raftutil_test

import (
	"io/ioutil"
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRaftutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Raftutil Suite")
}

var _ = BeforeSuite(func() {
	log.SetOutput(ioutil.Discard)
})
