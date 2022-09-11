package deployment_test

import (
	"io"
	"log"

	"github.com/james-lawrence/bw/internal/testingx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDeployment(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	log.SetOutput(io.Discard)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deployment Suite")
}

var _ = SynchronizedAfterSuite(func() {}, testingx.Cleanup)
