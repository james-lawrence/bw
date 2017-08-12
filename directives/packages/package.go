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
		if p, err := parse(pkg); err == nil {
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

	// Installs a set of packages
	InstallPackages(pkgs ...Package) error

	// RefreshCache refreshes the package cache.
	RefreshCache() error
}

// Install installs the provided packages.
func Install(tx transaction, pacs ...Package) error {
	var (
		err error
	)

	if len(pacs) == 0 {
		goto done
	}

	log.Println("refreshing cache")
	if err = tx.RefreshCache(); err != nil {
		err = errors.Wrap(err, "tx.RefreshCache failed")
		goto done
	}

	log.Println("installing packages")
	if err = tx.InstallPackages(pacs...); err != nil {
		err = errors.Wrap(err, "tx.IntallPackages failed")
		goto done
	}

done:
	return maybeCancel(tx, err)
}

func maybeCancel(tx transaction, err error) error {
	if err != nil {
		tx.Cancel()
		return err
	}

	return nil
}
