package astutil

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/james-lawrence/bw/internal/errorsx"
)

// ErrPackageNotFound returned when the requested package cannot be located
// within the given context.
const ErrPackageNotFound = errorsx.String("package not found")

// ErrAmbiguousPackage returned when the requested package is located multiple
// times within the given context.
const ErrAmbiguousPackage = errorsx.String("ambiguous package, found multiple matches within the provided context")

// ErrDeclarationNotFound returned when the requested declaration could not be located.
const ErrDeclarationNotFound = errorsx.String("declaration not found")

// ErrAmbiguousDeclaration returned when the requested declaration was located in multiple
// locations.
const ErrAmbiguousDeclaration = errorsx.String("ambiguous declaration, found multiple matches")

// ErrBasicLiteralNotFound returned when the requested literal could not be located.
const ErrBasicLiteralNotFound = errorsx.String("basic literal value not found")

// StrictPackageName only accepts packages that are an exact match.
func StrictPackageName(name string) func(*build.Package) bool {
	return func(pkg *build.Package) bool {
		return pkg.Name == name
	}
}

// LocatePackage finds a package by its name.
func LocatePackage(pkgName string, context build.Context, matches func(*build.Package) bool) (*build.Package, error) {
	pkg, err := context.Import(pkgName, ".", build.IgnoreVendor&build.ImportComment)
	_, noGoError := err.(*build.NoGoError)
	if err != nil && !noGoError {
		return nil, errors.Wrap(err, "failed to import the package")
	}

	if pkg != nil && (matches == nil || matches(pkg)) {
		return pkg, nil
	}

	return nil, ErrPackageNotFound
}

// ExtractFields walks the AST until it finds the first FieldList node.
// returns that node, If no node is found returns an empty FieldList.
func ExtractFields(decl ast.Spec) (list *ast.FieldList) {
	list = &ast.FieldList{}
	ast.Inspect(decl, func(n ast.Node) bool {
		if fields, ok := n.(*ast.FieldList); ok {
			list = fields
			return false
		}
		return true
	})
	return
}

type valueSpecFilter struct {
	specs []*ast.ValueSpec
}

func (t *valueSpecFilter) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.ValueSpec:
		t.specs = append(t.specs, n)
	}

	return t
}

// FindValueSpecs extracts all the ast.ValueSpec nodes from the provided tree.
func FindValueSpecs(node ast.Node) []*ast.ValueSpec {
	v := valueSpecFilter{}
	ast.Walk(&v, node)
	return v.specs
}

// FindConstants locates constants within the provided node's subtree.
func FindConstants(node ast.Node) []*ast.GenDecl {
	v := declFilter{keep: func(n *ast.GenDecl) bool {
		return n.Tok == token.CONST
	}}
	ast.Walk(&v, node)
	return v.nodes
}

// FindTypes locates type declarations within the provided node's subtree.
func FindTypes(node ast.Node) []*ast.GenDecl {
	v := declFilter{keep: func(n *ast.GenDecl) bool {
		return n.Tok == token.TYPE
	}}
	ast.Walk(&v, node)
	return v.nodes
}

// FindImports locates imports within the provided node's subtree.
func FindImports(node ast.Node) []*ast.GenDecl {
	v := declFilter{keep: func(n *ast.GenDecl) bool {
		return n.Tok == token.IMPORT
	}}
	ast.Walk(&v, node)
	return v.nodes
}

// GenDeclToDecl upcases GenDecl to Decl.
func GenDeclToDecl(decls ...*ast.GenDecl) (results []ast.Decl) {
	for _, d := range decls {
		results = append(results, d)
	}
	return results
}

type declFilter struct {
	nodes []*ast.GenDecl
	keep  func(*ast.GenDecl) bool
}

func (t *declFilter) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.GenDecl:
		if t.keep(n) {
			t.nodes = append(t.nodes, n)
		}
	}

	return t
}

type funcFilter struct {
	functions []*ast.FuncDecl
}

func (t *funcFilter) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return t
	}

	switch n := node.(type) {
	case *ast.FuncDecl:
		t.functions = append(t.functions, n)
	}

	return t
}

// FindFunc searches the provided node for ast.FuncDecl
func FindFunc(node ast.Node) []*ast.FuncDecl {
	v := funcFilter{}
	ast.Walk(&v, node)
	return v.functions
}

// SelectFuncDecl select a function from the set of function based on a filter.
func SelectFuncDecl(filter func(*ast.FuncDecl) bool, functions ...*ast.FuncDecl) []*ast.FuncDecl {
	r := make([]*ast.FuncDecl, 0, len(functions))
	for _, f := range functions {
		if filter(f) {
			r = append(r, f)
		}
	}
	return r
}

// SelectFuncType filters the provided GenDecl to ones that define functions.
func SelectFuncType(decls ...*ast.GenDecl) []*ast.GenDecl {
	result := make([]*ast.GenDecl, 0, len(decls))

	for _, decl := range decls {
		n := &ast.GenDecl{
			Tok:   decl.Tok,
			Doc:   decl.Doc,
			Specs: make([]ast.Spec, 0, len(decl.Specs)),
		}

		for _, s := range decl.Specs {
			if ts, ok := s.(*ast.TypeSpec); ok {
				if _, ok := ts.Type.(*ast.FuncType); ok {
					n.Specs = append(n.Specs, s)
				}
			}
		}

		result = append(result, n)
	}

	return result
}

// FilteredValue ...
type FilteredValue struct {
	Ident *ast.Ident
	Value ast.Expr
}

type valueFilter struct {
	values []FilteredValue
}

func (t *valueFilter) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.ValueSpec:
		for idx, name := range n.Names {
			t.values = append(t.values, FilteredValue{
				Ident: name,
				Value: n.Values[idx],
			})
		}
	}

	return t
}

// SelectValues searches the provided node for value specs.
func SelectValues(node ast.Node) []FilteredValue {
	v := valueFilter{}
	ast.Walk(&v, node)
	return v.values
}

// Searcher utility functions for searching for declarations within a set of packages.
type Searcher interface {
	FindFunction(f ast.Filter) (*ast.FuncDecl, error)
	FindUniqueType(f ast.Filter) (*ast.TypeSpec, error)
	FindFieldsForType(x ast.Expr) ([]*ast.Field, error)
}

// NewSearcher seatch the pkgset.
func NewSearcher(fset *token.FileSet, pkgset ...*build.Package) Searcher {
	return searcher{fset: fset, pkgset: pkgset}
}

type searcher struct {
	fset   *token.FileSet
	pkgset []*build.Package
}

func (t searcher) FindFunction(f ast.Filter) (*ast.FuncDecl, error) {
	return NewUtils(t.fset).FindFunction(f, t.pkgset...)
}

func (t searcher) FindUniqueType(f ast.Filter) (*ast.TypeSpec, error) {
	return NewUtils(t.fset).FindUniqueType(f, t.pkgset...)
}

func (t searcher) FindFieldsForType(x ast.Expr) ([]*ast.Field, error) {
	var (
		err  error
		spec *ast.TypeSpec
	)

	if spec, err = t.FindUniqueType(FilterName(types.ExprString(x))); err != nil {
		return []*ast.Field(nil), err
	}

	return ExtractFields(spec).List, nil
}

// Utils provides utility functions on packages.
type Utils interface {
	ParsePackages(pkgset ...*build.Package) ([]*ast.Package, error)
	FindUniqueType(f ast.Filter, packageSet ...*build.Package) (*ast.TypeSpec, error)
	FindFunction(f ast.Filter, pkgset ...*build.Package) (*ast.FuncDecl, error)
	WalkFiles(delegate func(path string, file *ast.File), pkgset ...*build.Package) error
}

// NewUtils ...
func NewUtils(fset *token.FileSet) Utils {
	return utils{fset: fset}
}

type utils struct {
	fset *token.FileSet
}

func (t utils) WalkFiles(delegate func(path string, file *ast.File), pkgset ...*build.Package) error {
	pkgs, err := t.ParsePackages(pkgset...)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		for p, f := range pkg.Files {
			delegate(p, f)
		}
	}

	return nil
}

func (t utils) ParsePackages(pkgset ...*build.Package) (result []*ast.Package, err error) {
	for _, pkg := range pkgset {
		filter := func(i os.FileInfo) bool {
			for _, f := range pkg.GoFiles {
				if f == i.Name() {
					return true
				}
			}

			return false
		}

		pkgs, err := parser.ParseDir(t.fset, pkg.Dir, filter, parser.ParseComments)
		if err != nil {
			return nil, err
		}

		for _, pkg := range pkgs {
			result = append(result, pkg)
		}
	}
	return result, nil
}

// FindUniqueType searches the provided packages for the unique declaration
// that matches the ast.Filter.
func (t utils) FindUniqueType(f ast.Filter, packageSet ...*build.Package) (*ast.TypeSpec, error) {
	pkgs, err := t.ParsePackages(packageSet...)
	if err != nil {
		return nil, err
	}

	found := FilterType(f, pkgs...)
	x := len(found)
	switch {
	case x == 0:
		return &ast.TypeSpec{}, ErrDeclarationNotFound
	case x == 1:
		return found[0], nil
	default:
		return &ast.TypeSpec{}, ErrAmbiguousDeclaration
	}
}

func (t utils) FindFunction(f ast.Filter, pkgset ...*build.Package) (*ast.FuncDecl, error) {
	pkgs, err := t.ParsePackages(pkgset...)
	if err != nil {
		return nil, err
	}

	found := []*ast.FuncDecl{}
	for _, pkg := range pkgs {
		found = append(found, FindFunc(pkg)...)
	}

	found = SelectFuncDecl(func(fx *ast.FuncDecl) bool { return f(fx.Name.Name) }, found...)
	x := len(found)
	switch {
	case x == 0:
		return &ast.FuncDecl{}, ErrDeclarationNotFound
	case x == 1:
		return found[0], nil
	default:
		return &ast.FuncDecl{}, ErrAmbiguousDeclaration
	}
}

// FilterType searches the provided packages for declarations that match
// the provided ast.Filter.
func FilterType(f ast.Filter, packageSet ...*ast.Package) []*ast.TypeSpec {
	types := []*ast.TypeSpec{}

	for _, pkg := range packageSet {
		ast.Inspect(pkg, func(n ast.Node) bool {
			typ, ok := n.(*ast.TypeSpec)
			if ok && f(typ.Name.Name) {
				types = append(types, typ)
			}

			return true
		})
	}

	return types
}

// FilterValue searches the provided packages for value specs that match
// the provided ast.Filter.
func FilterValue(f ast.Filter, packageSet ...*ast.Package) []*ast.ValueSpec {
	results := []*ast.ValueSpec{}

	for _, pkg := range packageSet {
		ast.Inspect(pkg, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.ValueSpec:
				results = append(results, x)
				return false
			case *ast.GenDecl:
				return ast.FilterDecl(x, f)
			default:
				return true
			}
		})
	}

	return results
}

// RetrieveBasicLiteralString searches the declarations for a literal string
// that matches the provided filter.
func RetrieveBasicLiteralString(f ast.Filter, packageSet ...*ast.Package) (string, error) {
	valueSpecs := FilterValue(f, packageSet...)
	switch len(valueSpecs) {
	case 0:
		// fallthrough
	case 1:
		valueSpec := valueSpecs[0]
		for idx, v := range valueSpec.Values {
			basicLit, ok := v.(*ast.BasicLit)
			if ok && basicLit.Kind == token.STRING && f(valueSpec.Names[idx].Name) {
				return strings.Trim(basicLit.Value, "`\""), nil
			}
		}
	default:
		return "", ErrAmbiguousDeclaration
	}

	return "", ErrBasicLiteralNotFound
}

// FilterName filter that matches the provided name by the name on a given node.
func FilterName(name string) ast.Filter {
	return func(in string) bool {
		return name == in
	}
}

// ASTPrinter convience printer that records the error that occurred.
// for later inspection.
type ASTPrinter struct {
	err error
}

// FprintAST prints the ast to the destination io.Writer.
func (t *ASTPrinter) FprintAST(dst io.Writer, fset *token.FileSet, ast interface{}) {
	if t.err == nil {
		t.err = printer.Fprint(dst, fset, ast)
	}
}

// Fprintln delegates to fmt.Fprintln, allowing for arbritrary text to be inlined.
func (t *ASTPrinter) Fprintln(dst io.Writer, a ...interface{}) {
	if t.err == nil {
		_, t.err = fmt.Fprintln(dst, a...)
	}
}

// Fprintf delegates to fmt.Fprintf, allowing for arbritrary text to be inlined.
func (t *ASTPrinter) Fprintf(dst io.Writer, format string, a ...interface{}) {
	if t.err == nil {
		_, t.err = fmt.Fprintf(dst, format, a...)
	}
}

// Err returns the recorded error, if any.
func (t *ASTPrinter) Err() error {
	return t.err
}

// // PrintPackage inserts the package and a preface at into the ast.
// func PrintPackage(printer ASTPrinter, dst io.Writer, fset *token.FileSet, pkg *build.Package, args []string) error {
// 	file := &ast.File{
// 		Name: &ast.Ident{
// 			Name: pkg.Name,
// 		},
// 	}
//
// 	printer.FprintAST(dst, fset, file)
// 	printer.Fprintf(dst, Preface, strings.Join(args, " "))
//
// 	// check if executed by go generate
// 	if os.Getenv("GOPACKAGE") != "" && os.Getenv("GOFILE") != "" && os.Getenv("GOLINE") != "" {
// 		printer.Fprintf(
// 			dst,
// 			"// invoked by go generate @ %s/%s line %s",
// 			os.Getenv("GOPACKAGE"),
// 			os.Getenv("GOFILE"),
// 			os.Getenv("GOLINE"),
// 		)
// 	}
//
// 	printer.Fprintf(dst, "\n\n")
//
// 	return errors.Wrap(printer.Err(), "failed to print the package header")
// }

// PrintDebug ...
func PrintDebug() string {
	var (
		buffer bytes.Buffer
	)

	if os.Getenv("GOPACKAGE") != "" && os.Getenv("GOFILE") != "" && os.Getenv("GOLINE") != "" {
		buffer.WriteString(
			fmt.Sprintf(
				"invoked by go generate @ %s/%s line %s\n",
				os.Getenv("GOPACKAGE"),
				os.Getenv("GOFILE"),
				os.Getenv("GOLINE"),
			),
		)
	}

	buffer.WriteString(strings.Join(os.Args[1:], " "))

	return buffer.String()
}
