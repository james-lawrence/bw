package bwfs_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBwfs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BWFS Suite")
}
