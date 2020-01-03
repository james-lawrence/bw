package notary_test

import (
	"github.com/golang/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/james-lawrence/bw/internal/x/testingx"

	. "github.com/james-lawrence/bw/notary"
)

var _ = Describe("Directory", func() {
	It("should be able to read/write/delete grants", func() {
		s := NewDirectory(
			testingx.TempDir(),
		)

		g1, err := s.Insert(Grant{
			Authorization: []byte{0},
		})
		Expect(err).To(Succeed())

		g1f, err := s.Lookup(g1.Fingerprint)
		Expect(err).To(Succeed())
		Expect(proto.Equal(&g1f, &g1)).To(BeTrue())

		g1d, err := s.Delete(g1)
		Expect(err).To(Succeed())
		Expect(proto.Equal(&g1d, &g1)).To(BeTrue())
	})

	It("should be able able to overwrite a grant", func() {
		s := NewDirectory(
			testingx.TempDir(),
		)

		g1, err := s.Insert(Grant{
			Authorization: []byte{0},
		})
		Expect(err).To(Succeed())
		g1u, err := s.Insert(Grant{
			Authorization: []byte{0},
		})
		Expect(err).To(Succeed())

		Expect(proto.Equal(&g1u, &g1)).To(BeTrue())
		g1f, err := s.Lookup(g1.Fingerprint)
		Expect(err).To(Succeed())
		Expect(proto.Equal(&g1f, &g1)).To(BeTrue())
		Expect(proto.Equal(&g1f, &g1u)).To(BeTrue())
	})
})
