package observers_test

import (
	"io/ioutil"
	"log"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestQuorum(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Observers Suite")
}

var _ = BeforeSuite(func() {
	log.SetOutput(ioutil.Discard)
})
