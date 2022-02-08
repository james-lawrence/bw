package bw_test

import (
	. "github.com/james-lawrence/bw"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func environmentSet1(k string) string {
	switch k {
	case "FOO":
		return "BAR"
	case "BIZZ":
		return "BAZZ"
	case "MULTILINE":
		return "line1\nline2\nline3"
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
		Entry("example 2",
			`field1: '${MULTILINE}'`,
			xType{
				Field1: "line1\nline2\nline3",
			},
		),
	)
})
