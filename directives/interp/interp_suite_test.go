package interp_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestInterp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Interp Suite")
}
