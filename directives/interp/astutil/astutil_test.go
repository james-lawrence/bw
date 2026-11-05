package astutil_test

import (
	"go/ast"
	"go/parser"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/james-lawrence/bw/directives/interp/astutil"
)

func TestExprTemplateList(t *testing.T) {
	t.Run("should work with no arguments", func(t *testing.T) {
		require.Empty(t, ExprTemplateList())
	})
}

func TestTypePattern(t *testing.T) {
	// build a query function based on the options
	for _, c := range []struct {
		name     string
		pattern  []ast.Expr
		provided []ast.Expr
		result   bool
	}{
		{
			"when expected length doesn't match the provided length return false",
			expr(),
			expr("int", "string"),
			false,
		},
		{
			"expect matching mattern of builtin types to return true",
			expr("int", "string"),
			expr("int", "string"),
			true,
		},
		{
			"package prefixed types should still match",
			expr("sql.Rows", "error"),
			expr("sql.Rows", "error"),
			true,
		},
		{
			"pointer types should still match",
			expr("*sql.Rows", "error"),
			expr("*sql.Rows", "error"),
			true,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.result, TypePattern(c.pattern...)(c.provided...))
		})
	}
}

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
