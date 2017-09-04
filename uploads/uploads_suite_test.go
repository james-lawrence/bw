package uploads_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestUploads(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Uploads Suite")
}
