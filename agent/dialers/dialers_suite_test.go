package dialers_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDialers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dialers Suite")
}
