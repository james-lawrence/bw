package bw_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/james-lawrence/bw"
)

var _ = Describe("bw", func() {
	Describe("SimpleGenerateID", func() {
		It("should generate different ids", func() {
			var (
				cache = make(map[string]struct{})
			)

			for i := 0; i < 100; i++ {
				id := MustGenerateID()
				_, found := cache[id.String()]
				Expect(found).To(BeFalse())
				cache[id.String()] = struct{}{}
			}
		})
	})
})
