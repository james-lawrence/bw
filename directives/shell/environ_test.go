package shell_test

import (
	"io/ioutil"
	"sort"

	"github.com/james-lawrence/bw/directives/shell"
	"github.com/joho/godotenv"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func mustReadString(path string) string {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	return string(raw)
}

func fromMap(m map[string]string) string {
	env, err := godotenv.Marshal(m)
	if err != nil {
		panic(err)
	}

	return env
}

var _ = Describe("Environ", func() {
	DescribeTable("Environment Parsing",
		func(example string, expected ...string) {
			result, err := shell.Environ(example)
			Expect(err).ToNot(HaveOccurred())
			sort.Strings(result)
			sort.Strings(expected)
			Expect(result).To(Equal(expected))
		},
		Entry("basic environment", "FOO=BAR", "FOO=BAR"),
		Entry("quoted environment", "FOO=\"HELLO WORLD\"", "FOO=\"HELLO WORLD\""),
		Entry("multiline environment 1", fromMap(map[string]string{"FOO": "HELLO\nWORLD"}), "FOO=\"HELLO\nWORLD\""),
		Entry("multiline environment 2", "FOO=\"HELLO\\nWORLD\"", "FOO=\"HELLO\nWORLD\""),
		Entry(
			"environmnet file example",
			mustReadString(".fixtures/environ.env"),
			"SIMPLE=VALUE",
			"QUOTED=\"QUOTED VALUE\"",
			"MULTILINE=\"Line1\nLine2\"",
		),
	)
})
