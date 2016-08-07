package deployment

import (
	"crypto/md5"
	"encoding/json"
	"log"
	"sort"

	"bitbucket.org/jatone/bearded-wookie/packagekit"
)

func NewDefaultCoordinator() (Coordinator, error) {
	pkgkit, err := packagekit.NewClient()
	if err != nil {
		return nil, err
	}

	return New(pkgkit), nil
}

// Builds a deployment Coordinator.
// Pass in a packagekit.Client implementation
func New(client packagekit.Client) Coordinator {
	return deployment{client}
}

type deployment struct {
	pkgkit packagekit.Client
}

// Status of the deployment Coordinator
func (t deployment) Status() error {
	return nil
}

func (t deployment) SystemStateChecksum() ([]byte, error) {
	var (
		hasher = md5.New()
	)

	packages, err := t.Packages()
	if err != nil {
		return nil, err
	}

	// sort to guarentee the ordering of the pages.
	sort.Sort(PackageByID(packages))

	json.NewEncoder(hasher).Encode(packages)

	return hasher.Sum(nil), nil
}

func (t deployment) Packages() ([]Package, error) {
	var (
		err                error
		tx                 packagekit.Transaction
		packages           []packagekit.Package
		exportablePackages []Package
	)

	if tx, err = t.pkgkit.CreateTransaction(); err != nil {
		return nil, err
	}

	if packages, err = tx.Packages(packagekit.FilterInstalled); err != nil {
		return nil, err
	}

	// convert into exported Type
	exportablePackages = make([]Package, 0, len(packages))
	for _, pkg := range packages {
		exportablePackages = append(exportablePackages, Package(pkg))
	}

	return exportablePackages, err
}

func (t deployment) InstallPackages(packageIDs ...string) error {
	var err error
	var tx packagekit.Transaction

	if tx, err = t.pkgkit.CreateTransaction(); err != nil {
		return err
	}
	defer tx.Cancel()

	// if err = tx.RefreshCache(); err != nil {
	// 	log.Println("tx.RefreshCache failed", err)
	// 	return err
	// }
	// if err = tx.DownloadPackages(true, packageIDs...); err != nil {
	// 	log.Println("tx.DownloadPackages failed", err)
	// 	return err
	// }

	if err = tx.InstallPackages(packageIDs...); err != nil {
		log.Println("tx.IntallPackages failed", err)
		return err
	}

	return nil
}
