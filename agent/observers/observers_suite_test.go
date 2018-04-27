package observers_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestQuorum(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Observers Suite")
}

var _ = BeforeSuite(func() {
	// log.SetOutput(ioutil.Discard)
})
