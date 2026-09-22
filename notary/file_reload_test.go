package notary_test

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/sshx"
	. "github.com/james-lawrence/bw/notary"
)

var _ = Describe("file storage reload", func() {
	It("should reload authorizations after repeated atomic replacement", func() {
		root := GinkgoT().TempDir()
		src := GinkgoT().TempDir()
		cfg := filepath.Join(src, "notary.yml")
		Expect(os.WriteFile(cfg, []byte("notary: {}\n"), 0600)).To(Succeed())

		c, err := NewFromFile(root, cfg)
		Expect(err).To(Succeed())

		newkey := func(comment string) (string, []byte) {
			s, err := QuickSigner()
			Expect(err).To(Succeed())
			fp, pub, err := s.AutoSignerInfo()
			Expect(err).To(Succeed())
			return fp, sshx.Comment(pub, comment)
		}

		fp1, pub1 := newkey("test1")
		fp2, pub2 := newkey("test2")

		swap := func(content []byte) {
			tmp := filepath.Join(src, "authorized_keys")
			Expect(os.WriteFile(tmp, content, 0600)).To(Succeed())
			Expect(CloneAuthorizationFile(tmp, filepath.Join(root, bw.AuthKeysFile))).To(Succeed())
		}

		swap(pub1)
		Eventually(func() error { _, err := c.Lookup(fp1); return err }, 2*time.Second, 50*time.Millisecond).Should(Succeed())

		swap(pub2)
		Eventually(func() error { _, err := c.Lookup(fp2); return err }, 2*time.Second, 50*time.Millisecond).Should(Succeed())
		Eventually(func() error { _, err := c.Lookup(fp1); return err }, 2*time.Second, 50*time.Millisecond).ShouldNot(Succeed())
	})
})
