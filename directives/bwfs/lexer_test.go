package bwfs

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func runLexer(l *lexer) []string {
	results := []string{}
	for t := l.NextToken(); t.typ != tokenFin; t = l.NextToken() {
		results = append(results, t.val)
	}
	return results
}

func lextest(t *testing.T, expected Archive, example string) {
	test := []string{
		expected.URI,
		expected.Path,
		prettyMode(expected.Mode),
		expected.Owner,
		expected.Group,
	}
	require.Equal(t, test, runLexer(lex(example)), fmt.Sprintf("%s : %s", expected, example))
}

func TestLexer(t *testing.T) {
	t.Run("lexing basic test cases", func(t *testing.T) {
		for i := 1; i <= 10; i++ {
			t.Run(fmt.Sprintf("example %d", i), func(t *testing.T) {
				a, encoded := archiveRandomExample()
				lextest(t, a, encoded)
			})
		}
	})

	t.Run("generated examples", func(t *testing.T) {
		t.Run("example 1 - leading/trailing whitespace", func(t *testing.T) {
			lextest(
				t,
				Archive{
					URI:   "s3://33cd:b41b:526c:7c4d:8532:b150:e8cd:c367?dolor=ipsum",
					Path:  "/root/path",
					Mode:  0777,
					Owner: "root",
					Group: "root",
				},
				" \t\r\n\"s3://33cd:b41b:526c:7c4d:8532:b150:e8cd:c367?dolor=ipsum\" \t\r\n /root/path 0777 root root",
			)
		})
	})

	t.Run("should return nothing on a comment", func(t *testing.T) {
		require.Empty(t, runLexer(lex(" // what a lovely bunch of comments\t\r\n")))
	})
}

func archiveRandomExample() (Archive, string) {
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
	return a, encode(a)
}
