package journal_test

import (
	"io"
	"log"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestJournal(t *testing.T) {
	log.SetOutput(io.Discard)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Journal Suite")
}
