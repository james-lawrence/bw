package bw_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/james-lawrence/bw"
)

var _ = Describe("bw", func() {
	Describe("SimpleGenerateID", func() {
		It("16 bytes", func() {
			id := MustGenerateID()
			Expect(id[:]).To(HaveLen(16))
		})

		It("should generate different ids", func() {
			var (
				cache = make(map[string]struct{})
			)

			for range 100 {
				id := MustGenerateID()
				_, found := cache[id.String()]
				Expect(found).To(BeFalse())
				cache[id.String()] = struct{}{}
			}
		})
	})
})
