package packages

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func runLexer(l *lexer) []string {
	results := []string{}
	for t := l.NextToken(); t.typ != tokenFin; t = l.NextToken() {
		results = append(results, t.val)
	}
	return results
}

var _ = Describe("Lexer", func() {
	DescribeTable("lexing",
		func(example string, expected ...string) {
			Expect(runLexer(lex(example))).To(Equal(expected))
		},
		Entry("example 1", `package`, "package"),
		Entry("example 2", `package;`, "package"),
		Entry("example 3", `'package'`, "package"),
		Entry("example 4", `;version;`, "", "version"),
		Entry("example 5", `'package'`, "package"),
		Entry("example 6", `'package';`, "package"),
		Entry("example 7", `package;version`, "package", "version"),
		Entry("example 8", `package;version;`, "package", "version"),
		Entry("example 9", `package;version;`, "package", "version"),
		Entry("example 10", `package;version;arch`, "package", "version", "arch"),
		Entry("example 11", `package;version;arch;repo`, "package", "version", "arch", "repo"),
		Entry("example 12", `'package';'version';'arch';'repo'`, "package", "version", "arch", "repo"),
		Entry("example 13", `package;version;;extra`, "package", "version", "", "extra"),
		Entry("example 14", `package;;;extra`, "package", "", "", "extra"),
		Entry("example 15", `'';;;extra`, "", "", "", "extra"),
	)
})
