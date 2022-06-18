package bootstrap_test

import (
	"io"
	"log"
	"testing"

	"github.com/james-lawrence/bw/internal/x/testingx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBootstrap(t *testing.T) {
	log.SetOutput(io.Discard)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bootstrap Suite")
}

var _ = SynchronizedAfterSuite(func() {}, testingx.Cleanup)
