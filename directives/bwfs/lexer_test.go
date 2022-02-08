package bwfs

import (
	"fmt"
	"math/rand"

	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

func runLexer(l *lexer) []string {
	results := []string{}
	for t := l.NextToken(); t.typ != tokenFin; t = l.NextToken() {
		results = append(results, t.val)
	}
	return results
}

func lextest(expected Archive, example string) {
	test := []string{
		expected.URI,
		expected.Path,
		prettyMode(expected.Mode),
		expected.Owner,
		expected.Group,
	}
	Expect(runLexer(lex(example))).To(Equal(test), fmt.Sprintf("%s : %s", expected, example))
}

var _ = Describe("Lexer", func() {
	DescribeTable("lexing basic test cases",
		lextest,
		archiveRandomExample("example 1"),
		archiveRandomExample("example 2"),
		archiveRandomExample("example 3"),
		archiveRandomExample("example 4"),
		archiveRandomExample("example 5"),
		archiveRandomExample("example 6"),
		archiveRandomExample("example 7"),
		archiveRandomExample("example 8"),
		archiveRandomExample("example 9"),
		archiveRandomExample("example 10"),
	)

	DescribeTable("generated examples",
		lextest,
		Entry(
			"example 1 - leading/trailing whitespace",
			Archive{
				URI:   "s3://33cd:b41b:526c:7c4d:8532:b150:e8cd:c367?dolor=ipsum",
				Path:  "/root/path",
				Mode:  0777,
				Owner: "root",
				Group: "root",
			},
			" \t\r\n\"s3://33cd:b41b:526c:7c4d:8532:b150:e8cd:c367?dolor=ipsum\" \t\r\n /root/path 0777 root root",
		),
	)

	It("should return nothing on a comment", func() {
		Expect(runLexer(lex(" // what a lovely bunch of comments\t\r\n"))).To(BeEmpty())
	})
})

func archiveRandomExample(desc string) TableEntry {
	encode := func(a Archive) string {
		sep := func() string {
			return stringFromCharset(rand.Intn(10)+1, []rune(CharsetSpace)...)
		}

		return fmt.Sprint(
			a.URI,
			sep(),
			a.Path,
			sep(),
			prettyMode(a.Mode),
			sep(),
			a.Owner,
			sep(),
			a.Group,
		)
	}

	a := randomArchive()
	return Entry(desc, a, encode(a))
}
