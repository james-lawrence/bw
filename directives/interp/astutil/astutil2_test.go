package astutil_test

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/require"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"testing"

	. "github.com/james-lawrence/bw/directives/interp/astutil"
)

func TestAstutil(t *testing.T) {
	t.Run("LocatePackage", func(t *testing.T) {
		t.Run("find a the specified package", func(t *testing.T) {
			var err error
			var p *build.Package

			p, err = LocatePackage("go/build", build.Default, StrictPackageName("build"))
			require.NoError(t, err)
			require.Equal(t, "build", p.Name)

			p, err = LocatePackage("does/not/exist", build.Default, StrictPackageName("exist"))
			require.Error(t, err)
			require.Nil(t, p)
		})
	})

	t.Run("ExtractFields", func(t *testing.T) {
		t.Run("should extract the fields from the provided ast.Spec", func(t *testing.T) {
			var err error
			var expr ast.Expr

			expr, err = parser.ParseExpr(emptyStructExpression)
			require.NoError(t, err)

			fields := ExtractFields(typeSpec("empty", expr))
			require.Empty(t, fields.List)

			expr, err = parser.ParseExpr(structExpression)
			require.NoError(t, err)

			fields = ExtractFields(typeSpec("fields", expr))
			require.Len(t, fields.List, 2)
		})
	})

	t.Run("FilterDeclarations", func(t *testing.T) {
		t.Run("should only locate declarations that match the filter", func(t *testing.T) {
			fset := token.NewFileSet()
			examples, err := parser.ParseFile(fset, "examples.go", examples, 0)
			require.NoError(t, err)

			p := ast.Package{
				Files: map[string]*ast.File{
					"examples.go": examples,
				},
			}

			types := FilterType(FilterName("aStruct"), &p)
			require.Len(t, types, 1)
			require.Equal(t, "aStruct", types[0].Name.Name)
		})

		t.Run("should be able to handle self referential types", func(t *testing.T) {
			fset := token.NewFileSet()
			examples, err := parser.ParseFile(fset, "examples.go", examples, 0)
			require.NoError(t, err)

			p := ast.Package{
				Files: map[string]*ast.File{
					"examples.go": examples,
				},
			}

			types := FilterType(FilterName("selfReferential"), &p)
			require.Len(t, types, 1)
			require.Equal(t, "selfReferential", types[0].Name.Name)
		})
	})

	t.Run("FindUniqueDeclaration", func(t *testing.T) {
		t.Run("should return the declaration if it is unique", func(t *testing.T) {
			fset := token.NewFileSet()
			p, err := LocatePackage("go/ast", build.Default, StrictPackageName("ast"))
			require.NoError(t, err)

			typ, err := NewUtils(fset).FindUniqueType(FilterName("Package"), p)
			require.NoError(t, err)
			require.Equal(t, "Package", typ.Name.Name)
		})

		t.Run("should return an error if the declaration is ambiguous", func(t *testing.T) {
			fset := token.NewFileSet()

			p, err := LocatePackage("go/ast", build.Default, StrictPackageName("ast"))
			require.NoError(t, err)

			typ, err := NewUtils(fset).FindUniqueType(FilterName("Package"), p, p)
			require.Equal(t, ErrAmbiguousDeclaration, err)
			require.Equal(t, &ast.TypeSpec{}, typ)
		})

		t.Run("should return an error if the declaration is not found", func(t *testing.T) {
			fset := token.NewFileSet()
			p, err := LocatePackage("go/ast", build.Default, StrictPackageName("ast"))
			require.NoError(t, err)

			typ, err := NewUtils(fset).FindUniqueType(FilterName("DoesNotExist"), p)
			require.Equal(t, ErrDeclarationNotFound, err)
			require.Equal(t, &ast.TypeSpec{}, typ)
		})
	})

	t.Run("FilterName", func(t *testing.T) {
		t.Run("should return true iff name matches exactly", func(t *testing.T) {
			filter := FilterName("aName")
			require.False(t, filter("OtherName"))
			require.True(t, filter("aName"))
		})
	})

	t.Run("ASTPrinter", func(t *testing.T) {
		t.Run("FprintAST", func(t *testing.T) {
			t.Run("should print the ast node into the buffer", func(t *testing.T) {
				pkg := &ast.File{
					Name: &ast.Ident{
						Name: "example",
					},
				}
				p := ASTPrinter{}
				fset := token.NewFileSet()
				dst := bytes.NewBuffer([]byte{})
				p.FprintAST(dst, fset, pkg)
				require.NoError(t, p.Err())
				require.Equal(t, "package example\n", dst.String())
			})
		})

		t.Run("Fprintln", func(t *testing.T) {
			t.Run("should print the provided elements into the buffer", func(t *testing.T) {
				p := ASTPrinter{}
				dst := bytes.NewBuffer([]byte{})
				p.Fprintln(dst, "Hello", "World")
				require.NoError(t, p.Err())
				require.Equal(t, "Hello World\n", dst.String())
			})
		})

		t.Run("Fprintf", func(t *testing.T) {
			t.Run("should print the formatted string into the buffer", func(t *testing.T) {
				p := ASTPrinter{}
				dst := bytes.NewBuffer([]byte{})
				p.Fprintf(dst, "Hello %s\n", "World")
				require.NoError(t, p.Err())
				require.Equal(t, "Hello World\n", dst.String())
			})
		})

		t.Run("Err", func(t *testing.T) {
			t.Run("should return the first error that occurred", func(t *testing.T) {
				p := ASTPrinter{}
				w1 := errWriter{err: fmt.Errorf("boom1")}
				w2 := errWriter{err: fmt.Errorf("boom2")}
				p.Fprintln(w1, "Hello World 1")
				p.Fprintln(w2, "Hello World 2")
				require.EqualError(t, p.Err(), "boom1")
			})
		})
	})

	t.Run("RetrieveBasicLiteralString", func(t *testing.T) {
		t.Run("should locate the value of the basic literal", func(t *testing.T) {
			fset := token.NewFileSet()
			examples, err := parser.ParseFile(fset, "examples.go", examples, 0)
			require.NoError(t, err)
			p := ast.Package{
				Files: map[string]*ast.File{
					"examples1.go": examples,
				},
			}

			value, err := RetrieveBasicLiteralString(FilterName("aConstant"), &p)
			require.NoError(t, err)
			require.Equal(t, "constant string", value)
		})

		t.Run("should return an error when the literal cannot be found", func(t *testing.T) {
			fset := token.NewFileSet()
			examples, err := parser.ParseFile(fset, "examples.go", examples, 0)
			require.NoError(t, err)
			p := ast.Package{
				Files: map[string]*ast.File{
					"examples1.go": examples,
				},
			}

			value, err := RetrieveBasicLiteralString(FilterName("aStruct"), &p)
			require.Equal(t, ErrBasicLiteralNotFound, err)
			require.Equal(t, "", value)
		})
	})
}

type errWriter struct {
	err error
}

func (t errWriter) Write([]byte) (int, error) {
	return 0, t.err
}

func typeSpec(name string, typ ast.Expr) ast.Spec {
	return &ast.TypeSpec{
		Name: &ast.Ident{Name: name},
		Type: typ,
	}
}

var examples = `package examples
type aStruct struct {
	Field1 int
	Field2 bool
}

type bStruct struct {
	// ensure there is a second declaration of aStruct.
	aStruct
}

type cStruct struct {
	Field1 int
	Field2 int
}

type selfReferential struct {
	Field1 int
	*selfReferential
}

type emptyStruct struct{}

const aConstant = "constant string"
`

var structExpression = `struct {
	Field1 int
	Field2 bool
}`

var emptyStructExpression = `struct{}`
