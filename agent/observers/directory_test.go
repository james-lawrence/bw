package observers_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw"
	. "github.com/james-lawrence/bw/agent/observers"
	"google.golang.org/grpc"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Directory", func() {
	It("should watch a directory for sockets", func() {
		dir, err := ioutil.TempDir(".", "directory")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(dir)
		dirconns, err := NewDirectory(dir)
		Expect(err).ToNot(HaveOccurred())

		addr := filepath.Join(dir, fmt.Sprintf("%s.sock", bw.MustGenerateID().String()))
		l, err := net.Listen("unix", addr)
		Expect(err).ToNot(HaveOccurred())
		gs := grpc.NewServer()
		go gs.Serve(l)
		Eventually(func() int { return dirconns.Observers() }).Should(Equal(1))
		gs.GracefulStop()
		Eventually(func() int { return dirconns.Observers() }).Should(Equal(0))
	})
})
