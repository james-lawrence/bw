package storage

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

const (
	s3protocolconfig = `bucket: example
key_prefix: foo
`
	localProtocolConfig = `directory: /tmp`
)

var _ = Describe("Uploads", func() {
	Describe("ProtocolFromConfig", func() {
		DescribeTable("protocol building",
			func(p string, c string, o interface{}) {
				r, err := ProtocolFromConfig(p, []byte(c))
				Expect(err).ToNot(HaveOccurred())
				Expect(r).To(BeAssignableToTypeOf(o))
			},
			Entry("s3 protocol", "s3", s3protocolconfig, s3P{}),
			Entry("file protocol", "file", localProtocolConfig, &Local{}),
		)

		It("should error our when provided an unknown protocol", func() {
			_, err := ProtocolFromConfig("does-not-exist", []byte(nil))
			Expect(err).To(MatchError("no protocol defined for: does-not-exist"))
		})
	})
})
