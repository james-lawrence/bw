package rsax_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestRsax(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rsax Suite")
}
