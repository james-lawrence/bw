package deployment

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
)

// PackagekitOption option for the Packagekit deployer.
type PackagekitOption func(*pkgkit) error

// PackagekitOptionPackageFilesDirectory loads all the package files within
// the specified directory.
func PackagekitOptionPackageFilesDirectory(dir string) PackagekitOption {
	return func(pkg *pkgkit) error {
		return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			var (
				f *os.File
			)
			if err != nil {
				return err
			}

			// skip sub directories.
			if info.IsDir() {
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

// NewPackagekit builds a coordinator that uses packagekit to install packages.
func NewPackagekit(options ...PackagekitOption) (Coordinator, error) {
	c := pkgkit{}

	for _, opt := range options {
		if err := opt(&c); err != nil {
			return nil, err
		}
	}

	return New(c), nil
}

type pkgkit struct {
	packages []string
}

func (t pkgkit) Deploy(completed chan error) error {
	log.Println("deploying")
	defer log.Println("deploy complete")
	log.Println("fetching latest from repositories")
	for _, p := range t.packages {
		log.Println("installing", p)
	}
	return nil
}
