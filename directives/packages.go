package directives

import (
	"context"
	"fmt"
	"os"

	"github.com/james-lawrence/bw/directives/packages"
	"github.com/james-lawrence/bw/packagekit"
	"github.com/pkg/errors"
)

// PackageLoader reads a set of packages to install from an io.Reader
type PackageLoader struct {
	Context
}

// Ext extensions to succeed against.
func (PackageLoader) Ext() []string {
	return []string{".bwpkg"}
}

// Load directive from file
func (t PackageLoader) Load(path string) (dir Directive, err error) {
	var (
		r    *os.File
		pkgs []packages.Package
	)

	if err = LoadsExtensions(path, "bwfs"); err != nil {
		return dir, err
	}

	if r, err = os.Open(path); err != nil {
		return dir, errors.WithStack(err)
	}
	defer r.Close()

	if pkgs, err = packages.ParseYAML(r); err != nil {
		return nil, err
	}

	if len(pkgs) == 0 {
		return closure(func(context.Context) error { return nil }), nil
	}

	return closure(func(context.Context) error {
		var (
			err error
			c   packagekit.Client
			tx  packagekit.Transaction
		)

		if c, tx, err = packagekit.NewTransaction(); err != nil {
			return err
		}
		defer c.Shutdown()

		if err = packages.RefreshCache(t.Context.Log, packagekitAdapter{tx}); err != nil {
			return err
		}

		if tx, err = c.CreateTransaction(); err != nil {
			return err
		}

		return packages.Install(t.Context.Log, packagekitAdapter{tx}, pkgs...)
	}), nil
}

type packagekitAdapter struct {
	packagekit.Transaction
}

func (t packagekitAdapter) Resolve(ctx context.Context, pacs ...packages.Package) (rpacs []packagekit.Package, err error) {
	spacs := make([]string, 0, len(pacs))
	for _, pac := range pacs {
		spacs = append(spacs, fmt.Sprintf("%s;%s;%s;%s", pac.Name, pac.Version, pac.Architecture, pac.Repository))
	}

	if rpacs, err = t.Transaction.Resolve(ctx, packagekit.FilterNone, spacs...); err != nil {
		return rpacs, err
	}

	return rpacs, err
}

func (t packagekitAdapter) InstallPackages(ctx context.Context, pacs ...packages.Package) (err error) {
	spacs := make([]packagekit.Package, 0, len(pacs))
	for _, pac := range pacs {
		spacs = append(spacs, packagekit.Package{
			ID: fmt.Sprintf("%s;%s;%s;%s", pac.Name, pac.Version, pac.Architecture, pac.Repository),
		})
	}

	options := packagekit.TransactionFlagNone | packagekit.TransactionFlagAllowDowngrade | packagekit.TransactionFlagAllowReinstall
	return t.Transaction.InstallPackages(ctx, options, spacs...)
}
