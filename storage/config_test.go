package storage

import (
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	yaml "gopkg.in/yaml.v1"
)

var _ = Describe("Config", func() {
	const (
		s3OptionsConfig = `backend: s3
options:
  bucket: example
  key_prefix: foo
`
		s3OptionsOutput = `bucket: example
key_prefix: foo
`
	)

	DescribeTable("options parsing", func(in string, out string) {
		var (
			c Config
		)

		Expect(yaml.Unmarshal([]byte(in), &c)).ToNot(HaveOccurred())
		opts, err := c.options()
		log.Println("config", c)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(opts)).To(BeEquivalentTo(out))
	},
		Entry("s3 options example", s3OptionsConfig, s3OptionsOutput),
	)
})
