package shell_test

import (
	"io/ioutil"
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestShell(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Shell Suite")
}

var _ = BeforeSuite(func() {
	log.SetOutput(ioutil.Discard)
})