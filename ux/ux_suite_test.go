package ux_test

import (
	"context"
	"io"
	"log"
	"syscall"
	"testing"

	"github.com/james-lawrence/bw/internal/x/debugx"
	"github.com/james-lawrence/bw/internal/x/testingx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUx(t *testing.T) {
	go debugx.DumpOnSignal(context.Background(), syscall.SIGUSR2)
	log.SetOutput(io.Discard)
	RegisterFailHandler(Fail)
	RunSpecs(t, "UX Suite")
}

var _ = SynchronizedAfterSuite(func() {}, testingx.Cleanup)
