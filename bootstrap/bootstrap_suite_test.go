package bootstrap_test

import (
	"io/ioutil"
	"log"
	"testing"

	"github.com/james-lawrence/bw/internal/x/testingx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBootstrap(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bootstrap Suite")
}

var _ = SynchronizedAfterSuite(func() {}, func() {
	testingx.Cleanup()
})
