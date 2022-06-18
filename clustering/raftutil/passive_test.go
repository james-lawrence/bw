package raftutil

import (
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("passive state", func() {
	Context("stable", func() {
		It("should not modify the current state", func() {
			p := passive{
				failures: 10,
				protocol: &Protocol{},
				sgroup:   &sync.WaitGroup{},
			}
			updated := p.stable()
			Expect(p.failures).To(Equal(uint64(10)))
			Expect(updated.next.(passive).failures).To(Equal(uint64(0)))
		})
	})

	Context("unstable", func() {
		It("should not modify the current state", func() {
			p := passive{
				failures: 10,
				protocol: &Protocol{},
				sgroup:   &sync.WaitGroup{},
			}
			updated := p.unstable()
			Expect(p.failures).To(Equal(uint64(10)))
			Expect(updated.next.(passive).failures).To(Equal(uint64(11)))
		})
	})
})
