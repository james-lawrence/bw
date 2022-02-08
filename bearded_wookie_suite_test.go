package bw_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBeardedWookie(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BeardedWookie Suite")
}
