package astutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAstutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Astutil Suite")
}
