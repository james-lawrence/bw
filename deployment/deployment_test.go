package deployment_test

import (
	. "bitbucket.org/jatone/bearded-wookie/deployment"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Deployment", func() {
	Describe("PackageByID", func() {
		Describe("Len", func() {
			It("returns the number of Packages in the Array", func() {
				subject := PackageByID{Package{ID: "1"}, Package{ID: "2"}}
				Expect(subject.Len()).To(Equal(2))
			})
		})

		Describe("Less", func() {
			It("returns true if the package at index i has a lower Id than the package at index j", func() {
				subject := PackageByID{Package{ID: "1"}, Package{ID: "2"}}
				Expect(subject.Less(0, 1)).To(BeTrue())
			})

			It("returns false if the package at index i has an Id greater than the package at index j", func() {
				subject := PackageByID{Package{ID: "3"}, Package{ID: "2"}}
				Expect(subject.Less(0, 1)).To(BeFalse())
			})
		})

		Describe("Swap", func() {
			It("swaps the position of the packages at indexes i and j in the Array", func() {
				subject := PackageByID{Package{ID: "1"}, Package{ID: "2"}}
				Expect(subject[0].ID).To(Equal("1"))
				Expect(subject[1].ID).To(Equal("2"))

				subject.Swap(0, 1)

				Expect(subject[0].ID).To(Equal("2"))
				Expect(subject[1].ID).To(Equal("1"))
			})
		})
	})
})
