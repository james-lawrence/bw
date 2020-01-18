package astutil_test

import (
	"go/ast"
	"go/parser"

	. "github.com/james-lawrence/bw/directives/interp/astutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Astutil", func() {
	Describe("ExprTemplateList", func() {
		It("should work with no arguments", func() {
			Expect(ExprTemplateList()).To(BeEmpty())
		})
	})

	Describe("TypePattern", func() {
		DescribeTable("build a query function based on the options",
			func(pattern, provided []ast.Expr, result bool) {
				Expect(TypePattern(pattern...)(provided...)).To(Equal(result))
			},
			Entry(
				"when expected length doesn't match the provided length return false",
				expr(),
				expr("int", "string"),
				false,
			),
			Entry(
				"expect matching mattern of builtin types to return true",
				expr("int", "string"),
				expr("int", "string"),
				true,
			),
			Entry(
				"expect matching mattern of builtin types to return true",
				expr("int", "string"),
				expr("int", "string"),
				true,
			),
			Entry(
				"package prefixed types should still match",
				expr("sql.Rows", "error"),
				expr("sql.Rows", "error"),
				true,
			),
			Entry(
				"pointer types should still match",
				expr("*sql.Rows", "error"),
				expr("*sql.Rows", "error"),
				true,
			),
		)
	})
})

func expr(sl ...string) []ast.Expr {
	xl := make([]ast.Expr, 0, len(sl))
	for _, s := range sl {
		x, err := parser.ParseExpr(s)
		if err != nil {
			panic(err)
		}
		xl = append(xl, x)
	}
	return xl
}
