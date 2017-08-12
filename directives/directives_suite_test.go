package directives_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDirectives(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Directives Suite")
}
