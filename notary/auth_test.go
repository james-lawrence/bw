package notary

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/james-lawrence/bw/internal/x/testingx"
)

var _ = Describe("newAutoSignerPath", func() {
	It("should succeed when no key exists", func() {
		_, err := newAutoSignerPath(
			filepath.Join(testingx.TempDir(), bw.DefaultNotaryKey),
			"",
			rsax.UnsafeAuto,
		)
		Expect(err).To(Succeed())
	})

	It("should fail when unable to write to disk", func() {
		tmp := testingx.TempDir()
		os.RemoveAll(tmp)
		_, err := newAutoSignerPath(
			filepath.Join(tmp, bw.DefaultNotaryKey),
			"",
			rsax.UnsafeAuto,
		)
		Expect(err).ToNot(Succeed())
	})
})
