package deployment_test

import (
	"io/ioutil"
	"log"

	"github.com/james-lawrence/bw/internal/x/testingx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDeployment(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deployment Suite")
}

var _ = SynchronizedAfterSuite(func() {}, testingx.Cleanup)
