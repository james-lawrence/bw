package astutil_test

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"

	. "github.com/james-lawrence/bw/directives/interp/astutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Astutil", func() {
	Describe("LocatePackage", func() {
		It("find a the specified package", func() {
			var err error
			var p *build.Package

			p, err = LocatePackage("go/build", build.Default, StrictPackageName("build"))
			Expect(err).ToNot(HaveOccurred())
			Expect(p.Name).To(Equal("build"))

			p, err = LocatePackage("does/not/exist", build.Default, StrictPackageName("exist"))
			Expect(err).To(HaveOccurred())
			Expect(p).To(BeNil())
		})
	})

	Describe("ExtractFields", func() {
		It("should extract the fields from the provided ast.Spec", func() {
			var err error
			var expr ast.Expr

			expr, err = parser.ParseExpr(emptyStructExpression)
			Expect(err).ToNot(HaveOccurred())

			fields := ExtractFields(typeSpec("empty", expr))
			Expect(fields.List).To(BeEmpty())

			expr, err = parser.ParseExpr(structExpression)
			Expect(err).ToNot(HaveOccurred())

			fields = ExtractFields(typeSpec("fields", expr))
			Expect(fields.List).To(HaveLen(2))
		})
	})

	Describe("FilterDeclarations", func() {
		It("should only locate declarations that match the filter", func() {
			fset := token.NewFileSet()
			examples, err := parser.ParseFile(fset, "examples.go", examples, 0)
			Expect(err).ToNot(HaveOccurred())

			p := ast.Package{
				Files: map[string]*ast.File{
					"examples.go": examples,
				},
			}

			types := FilterType(FilterName("aStruct"), &p)
			Expect(types).To(HaveLen(1))
			Expect(types[0].Name.Name).To(Equal("aStruct"))
		})

		It("should be able to handle self referential types", func() {
			fset := token.NewFileSet()
			examples, err := parser.ParseFile(fset, "examples.go", examples, 0)
			Expect(err).ToNot(HaveOccurred())

			p := ast.Package{
				Files: map[string]*ast.File{
					"examples.go": examples,
				},
			}

			types := FilterType(FilterName("selfReferential"), &p)
			Expect(types).To(HaveLen(1))
			Expect(types[0].Name.Name).To(Equal("selfReferential"))
		})
	})

	Describe("FindUniqueDeclaration", func() {
		It("should return the declaration if it is unique", func() {
			fset := token.NewFileSet()
			p, err := LocatePackage("go/ast", build.Default, StrictPackageName("ast"))
			Expect(err).ToNot(HaveOccurred())

			typ, err := NewUtils(fset).FindUniqueType(FilterName("Package"), p)
			Expect(err).ToNot(HaveOccurred())
			Expect(typ.Name.Name).To(Equal("Package"))
		})

		It("should return an error if the declaration is ambiguous", func() {
			fset := token.NewFileSet()

			p, err := LocatePackage("go/ast", build.Default, StrictPackageName("ast"))
			Expect(err).ToNot(HaveOccurred())

			typ, err := NewUtils(fset).FindUniqueType(FilterName("Package"), p, p)
			Expect(err).To(Equal(ErrAmbiguousDeclaration))
			Expect(typ).To(Equal(&ast.TypeSpec{}))
		})

		It("should return an error if the declaration is not found", func() {
			fset := token.NewFileSet()
			p, err := LocatePackage("go/ast", build.Default, StrictPackageName("ast"))
			Expect(err).ToNot(HaveOccurred())

			typ, err := NewUtils(fset).FindUniqueType(FilterName("DoesNotExist"), p)
			Expect(err).To(Equal(ErrDeclarationNotFound))
			Expect(typ).To(Equal(&ast.TypeSpec{}))
		})
	})

	Describe("FilterName", func() {
		It("should return true iff name matches exactly", func() {
			filter := FilterName("aName")
			Expect(filter("OtherName")).To(BeFalse())
			Expect(filter("aName")).To(BeTrue())
		})
	})

	Describe("ASTPrinter", func() {
		Describe("FprintAST", func() {
			It("should print the ast node into the buffer", func() {
				pkg := &ast.File{
					Name: &ast.Ident{
						Name: "example",
					},
				}
				p := ASTPrinter{}
				fset := token.NewFileSet()
				dst := bytes.NewBuffer([]byte{})
				p.FprintAST(dst, fset, pkg)
				Expect(p.Err()).ToNot(HaveOccurred())
				Expect(dst.String()).To(Equal("package example\n"))
			})
		})

		Describe("Fprintln", func() {
			It("should print the provided elements into the buffer", func() {
				p := ASTPrinter{}
				dst := bytes.NewBuffer([]byte{})
				p.Fprintln(dst, "Hello", "World")
				Expect(p.Err()).ToNot(HaveOccurred())
				Expect(dst.String()).To(Equal("Hello World\n"))
			})
		})

		Describe("Fprintf", func() {
			It("should print the formatted string into the buffer", func() {
				p := ASTPrinter{}
				dst := bytes.NewBuffer([]byte{})
				p.Fprintf(dst, "Hello %s\n", "World")
				Expect(p.Err()).ToNot(HaveOccurred())
				Expect(dst.String()).To(Equal("Hello World\n"))
			})
		})

		Describe("Err", func() {
			It("should return the first error that occurred", func() {
				p := ASTPrinter{}
				w1 := errWriter{err: fmt.Errorf("boom1")}
				w2 := errWriter{err: fmt.Errorf("boom2")}
				p.Fprintln(w1, "Hello World 1")
				p.Fprintln(w2, "Hello World 2")
				Expect(p.Err()).To(MatchError("boom1"))
			})
		})
	})

	Describe("RetrieveBasicLiteralString", func() {
		It("should locate the value of the basic literal", func() {
			fset := token.NewFileSet()
			examples, err := parser.ParseFile(fset, "examples.go", examples, 0)
			Expect(err).ToNot(HaveOccurred())
			p := ast.Package{
				Files: map[string]*ast.File{
					"examples1.go": examples,
				},
			}

			value, err := RetrieveBasicLiteralString(FilterName("aConstant"), &p)
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal("constant string"))
		})

		It("should return an error when the literal cannot be found", func() {
			fset := token.NewFileSet()
			examples, err := parser.ParseFile(fset, "examples.go", examples, 0)
			Expect(err).ToNot(HaveOccurred())
			p := ast.Package{
				Files: map[string]*ast.File{
					"examples1.go": examples,
				},
			}

			value, err := RetrieveBasicLiteralString(FilterName("aStruct"), &p)
			Expect(err).To(Equal(ErrBasicLiteralNotFound))
			Expect(value).To(Equal(""))
		})
	})
})

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
