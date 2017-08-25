package bw_test

import (
	. "bitbucket.org/jatone/bearded-wookie"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func environmentSet1(k string) string {
	switch k {
	case "FOO":
		return "BAR"
	case "BIZZ":
		return "BAZZ"
	default:
		return ""
	}
}

type xType struct {
	Field1 string
	Field2 string
	Field3 string
}

var _ = Describe("Config", func() {
	DescribeTable("ExpandEnvironAndDecode", func(content string, result xType) {
		out := xType{}
		Expect(ExpandEnvironAndDecode([]byte(content), &out, environmentSet1)).ToNot(HaveOccurred())
		Expect(out).To(Equal(result))
	},
		Entry("example 1",
			`field1: "${FOO}"
field2: "${BIZZ}"
field3: "YOINK"
`,
			xType{
				Field1: "BAR",
				Field2: "BAZZ",
				Field3: "YOINK",
			},
		),
	)
})
