package directives

import (
	"fmt"
	"io"
	"log"

	"bitbucket.org/jatone/bearded-wookie/directives/packages"
	"bitbucket.org/jatone/bearded-wookie/packagekit"
)

// PackageLoader reads a set of packages to install from an io.Reader
type PackageLoader struct{}

// Ext extensions to succeed against.
func (PackageLoader) Ext() []string {
	return []string{".bwpkg"}
}

// Build builds a directive from the reader.
func (t PackageLoader) Build(r io.Reader) (Directive, error) {
	var (
		err  error
		pkgs []packages.Package
	)

	if pkgs, err = packages.ParseYAML(r); err != nil {
		return nil, err
	}

	if len(pkgs) == 0 {
		return closure(func() error { return nil }), nil
	}

	return closure(func() error {
		var (
			err error
			c   packagekit.Client
			tx  packagekit.Transaction
		)
		log.Println("--------------------- PackageKit Transaction")
		if c, tx, err = packagekit.NewTransaction(); err != nil {
			return err
		}
		defer c.Shutdown()

		return packages.Install(packagekitAdapter{tx}, pkgs...)
	}), nil
}

type packagekitAdapter struct {
	packagekit.Transaction
}

func (t packagekitAdapter) InstallPackages(pacs ...packages.Package) error {
	spacs := make([]string, 0, len(pacs))
	for _, pac := range pacs {
		spacs = append(spacs, fmt.Sprintf("%s;%s;%s;%s", pac.Name, pac.Version, pac.Architecture, pac.Repository))
	}

	return t.Transaction.InstallPackages(spacs...)
}
