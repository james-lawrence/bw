package packages

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"time"

	"github.com/james-lawrence/bw/packagekit"
	"github.com/pkg/errors"

	yaml "gopkg.in/yaml.v2"
)

// Package represents a package directive.
// Packages follow the format name;version;arch;repo
// all fields are optional except the name.
type Package struct {
	Name         string
	Version      string
	Architecture string
	Repository   string
}

// ParseYAML parses package directives from a YAML source.
func ParseYAML(r io.Reader) ([]Package, error) {
	var (
		err     error
		raw     []byte
		decoded []string
		results []Package
	)

	if raw, err = ioutil.ReadAll(r); err != nil {
		return results, errors.Wrap(err, "failed to read yaml")
	}

	if err = yaml.Unmarshal(raw, &decoded); err != nil {
		return results, errors.Wrap(err, "failed to decode ymal")
	}

	results = make([]Package, 0, len(decoded))
	for _, pkg := range decoded {
		if p, err := Parse(pkg); err == nil {
			results = append(results, p)
		} else {
			return results, errors.Wrapf(err, "failed to parse package directive: %s", pkg)
		}
	}

	return results, nil
}

type transaction interface {
	// Cancel this transaction.
	Cancel() error

	// Resolve a set of packages
	Resolve(ctx context.Context, pkgs ...Package) ([]packagekit.Package, error)

	// Installs a set of packages
	InstallPackages(ctx context.Context, pkgs ...Package) error

	// RefreshCache refreshes the package cache.
	RefreshCache(context.Context) (time.Duration, error)
}

// RefreshCache ...
func RefreshCache(l logger, tx transaction) (err error) {
	var (
		d time.Duration
	)

	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer done()

	l.Println("------------------- initiated cache refresh -------------------")
	if d, err = tx.RefreshCache(ctx); err != nil {
		goto done
	}
	l.Printf("cache refresh duration: %s\n", d)
	l.Println("------------------- completed cache refresh -------------------")
done:
	return maybeCancel(tx, errors.Wrap(err, "failed cache refresh"))
}

// Resolve ...
func Resolve(l logger, tx transaction, pacs ...Package) (pset []packagekit.Package, err error) {
	if len(pacs) == 0 {
		return pset, tx.Cancel()
	}

	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer done()
	l.Println("------------------- initiated package resolution -------------------")
	if pset, err = tx.Resolve(ctx, pacs...); err != nil {
		goto done
	}
	l.Println("------------------- completed package resolution -------------------")
done:
	return pset, maybeCancel(tx, errors.Wrap(err, "failed package resolution"))
}

// Install installs the provided packages.
func Install(l logger, tx transaction, pacs ...Package) (err error) {
	if len(pacs) == 0 {
		return tx.Cancel()
	}

	ctx, done := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer done()
	l.Println("------------------- initiated package installation -------------------")
	if err = errors.Wrap(tx.InstallPackages(ctx, pacs...), "failed package installation"); err != nil {
		goto done
	}
	l.Println("------------------- completed package installation -------------------")
done:
	return maybeCancel(tx, errors.Wrap(err, "failed package installation"))
}

func maybeCancel(tx transaction, err error) error {
	if err != nil {
		tx.Cancel()
		log.Println(err)
		return err
	}

	return nil
}

type logger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})
}
