package muxer_test

import (
	"io"
	"log"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/james-lawrence/bw/internal/testingx"
)

func Test(t *testing.T) {
	log.SetOutput(io.Discard)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Muxer Suite")
}

var _ = SynchronizedAfterSuite(func() {}, testingx.Cleanup)
