package packages

import (
	"io"
	"io/ioutil"
	"log"

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
	Resolve(pkgs ...Package) ([]Package, error)

	// Installs a set of packages
	InstallPackages(pkgs ...Package) error

	// RefreshCache refreshes the package cache.
	RefreshCache() error
}

// RefreshCache ...
func RefreshCache(tx transaction) (err error) {
	log.Println("------------------- refreshing cache")
	if err = errors.Wrap(tx.RefreshCache(), "tx.RefreshCache failed"); err != nil {
		goto done
	}

done:
	return maybeCancel(tx, err)
}

// Resolve ...
func Resolve(tx transaction, pacs ...Package) (pset []Package, err error) {
	if len(pacs) == 0 {
		return pset, tx.Cancel()
	}

	log.Println("------------------- refreshing cache")
	if err = errors.Wrap(tx.RefreshCache(), "tx.RefreshCache failed"); err != nil {
		goto done
	}

	log.Println("------------------- resolving packages")
	if pset, err = tx.Resolve(pacs...); err != nil {
		err = errors.Wrap(err, "tx.Resolve failed")
		goto done
	}

done:
	return pset, maybeCancel(tx, err)
}

// Install installs the provided packages.
func Install(tx transaction, pacs ...Package) error {
	var (
		err error
	)

	if len(pacs) == 0 {
		return tx.Cancel()
	}

	log.Println("------------------- installing packages")
	if err = errors.Wrap(tx.InstallPackages(pacs...), "tx.IntallPackages failed"); err != nil {
		goto done
	}

done:
	return maybeCancel(tx, err)
}

func maybeCancel(tx transaction, err error) error {
	if err != nil {
		log.Println("package installation failed", err)
		tx.Cancel()
		return err
	}

	return nil
}
