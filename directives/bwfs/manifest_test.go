package bwfs

import (
	"io"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest", func() {
	default1 := Archive{
		URI:   "s3://33cd:b41b:526c:7c4d:8532:b150:e8cd:c367?dolor=ipsum",
		Path:  "/root/path",
		Mode:  0755,
		Owner: "root",
		Group: "root",
	}

	DescribeTable("parsing examples",
		func(src io.Reader, d Archive, expected ...Archive) {
			results, err := ParseManifest(d, src)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(Equal(expected))
		},
		Entry(
			"example 1 - parse one per line",
			strings.NewReader(default1.String()+"\n"+default1.String()+"\n"),
			default1,
			default1,
			default1,
		),
	)
})
