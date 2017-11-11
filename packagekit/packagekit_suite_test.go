package packagekit_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPackagekit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Packagekit Suite")
}

var _ = BeforeSuite(func() {
	// log.SetOutput(ioutil.Discard)
})
