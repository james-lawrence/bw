package ux_test

import (
	"context"
	"log"
	"os"
	"syscall"
	"testing"

	"github.com/james-lawrence/bw/internal/x/debugx"
	"github.com/james-lawrence/bw/internal/x/testingx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestUx(t *testing.T) {
	log.Println("PID", os.Getpid())
	go debugx.DumpOnSignal(context.Background(), syscall.SIGUSR2)
	// log.SetOutput(ioutil.Discard)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ux Suite")
}

var _ = SynchronizedAfterSuite(func() {}, func() {
	testingx.Cleanup()
})
