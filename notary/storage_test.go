package notary_test

import (
	"bytes"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/james-lawrence/bw/internal/sshx"
	. "github.com/james-lawrence/bw/notary"
)

var _ = Describe("ReplaceAuthorizedKey", func() {
	It("should allow repeated replace calls without change", func() {
		auths, err := ioutil.TempFile(GinkgoT().TempDir(), "auths.tmp")
		Expect(err).To(Succeed())

		s1, err := QuickSigner()
		Expect(err).To(Succeed())
		fp1, pubk1, err := s1.AutoSignerInfo()
		Expect(err).To(Succeed())
		Expect(ReplaceAuthorizedKey(auths.Name(), fp1, sshx.Comment(pubk1, "test1"))).To(Succeed())

		s2, err := QuickSigner()
		Expect(err).To(Succeed())
		fp2, pubk2, err := s2.AutoSignerInfo()
		Expect(err).To(Succeed())
		Expect(ReplaceAuthorizedKey(auths.Name(), fp2, sshx.Comment(pubk2, "test2"))).To(Succeed())
		Expect(ReplaceAuthorizedKey(auths.Name(), fp2, sshx.Comment(pubk2, "test2"))).To(Succeed())

		content, err := ioutil.ReadFile(auths.Name())
		Expect(err).To(Succeed())
		Expect(string(content)).To(Equal(
			string(bytes.Join(
				[][]byte{
					sshx.Comment(pubk1, "test1"),
					sshx.Comment(pubk2, "test2"),
				},
				[]byte{},
			)),
		))
	})
})
