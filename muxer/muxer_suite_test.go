package muxer_test

import (
	"io"
	"log"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/james-lawrence/bw/internal/x/testingx"
)

func Test(t *testing.T) {
	log.SetOutput(io.Discard)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Muxer Suite")
}

var _ = SynchronizedAfterSuite(func() {}, testingx.Cleanup)