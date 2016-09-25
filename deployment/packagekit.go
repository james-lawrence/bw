package deployment

import (
	"bufio"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"bitbucket.org/jatone/bearded-wookie/packagekit"
)

// PackagekitOption option for the Packagekit deployer.
type PackagekitOption func(*pkgkit) error

// PackagekitOptionPackageFilesDirectory loads all the package files within
// the specified directory.
func PackagekitOptionPackageFilesDirectory(dir string) PackagekitOption {
	return func(pkg *pkgkit) error {
		log.Println("walking", dir)
		return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			var (
				f *os.File
			)
			log.Println("processing", path)
			if err != nil {
				return err
			}

			// skip sub directories.
			if info.IsDir() && path != dir {
				return filepath.SkipDir
			}

			if f, err = os.Open(path); err != nil {
				return err
			}

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				pkg.packages = append(pkg.packages, scanner.Text())
			}

			return nil
		})
	}
}

func connect(pkgkit *pkgkit) error {
	var (
		err error
	)

	pkgkit.client, err = packagekit.NewClient()

	return err
}

// NewPackagekit builds a coordinator that uses packagekit to install packages.
func NewPackagekit(options ...PackagekitOption) Coordinator {
	coord := pkgkit{
		options: append(options, connect),
	}

	return New(coord)
}

type pkgkit struct {
	options  []PackagekitOption
	client   packagekit.Client
	packages []string
}

func (t pkgkit) Deploy(completed chan error) error {
	for _, opt := range t.options {
		if err := opt(&t); err != nil {
			return err
		}
	}

	go t.deploy(completed)
	return nil
}

func (t pkgkit) deploy(completed chan error) {
	var (
		err error
		tx  packagekit.Transaction
	)
	log.Println("deploying")
	defer log.Println("deploy complete")

	if tx, err = t.client.CreateTransaction(); err != nil {
		err = errors.Wrap(err, "failed to created transaction")
		goto done
	}

	log.Println("refreshing cache")
	if err = tx.RefreshCache(); err != nil {
		err = errors.Wrap(err, "tx.RefreshCache failed")
		goto done
	}

	if tx, err = t.client.CreateTransaction(); err != nil {
		err = errors.Wrap(err, "failed to created transaction")
		goto done
	}

	log.Println("installing packages")
	if err = tx.InstallPackages(t.packages...); err != nil {
		err = errors.Wrap(err, "tx.IntallPackages failed")
		goto done
	}

done:
	if err != nil {
		tx.Cancel()
		log.Println(err)
	}
	completed <- err
}
