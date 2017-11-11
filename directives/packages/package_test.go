package packages_test

import (
	"strings"

	. "github.com/james-lawrence/bw/directives/packages"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Package", func() {
	const yaml1 = `
- vim;0.0.0;x86_64;extra
- htop;0.0.0;;extra
- strace
`
	const yaml2 = `
- strace
`
	DescribeTable("ParseYAML",
		func(example string, expected ...Package) {
			packages, err := ParseYAML(strings.NewReader(example))
			Expect(err).ToNot(HaveOccurred())
			Expect(packages).To(Equal(expected))
		},
		Entry(
			"example 1", yaml1,
			Package{
				Name:         "vim",
				Version:      "0.0.0",
				Architecture: "x86_64",
				Repository:   "extra",
			},
			Package{
				Name:       "htop",
				Version:    "0.0.0",
				Repository: "extra",
			},
			Package{
				Name: "strace",
			},
		),
		Entry(
			"example 2", yaml2,
			Package{
				Name: "strace",
			},
		),
	)
})
