package packagekit

import (
	"math/rand"
	"strings"

	"github.com/manveru/faker"
)

// NewDummyClient - Returns a new dummy Client for testing purposes.
func NewDummyClient(packageList ...Package) Client {
	return dummyClient{packageList}
}

// NewDummyTransaction - Returns a new dummy Transaction for testing purposes.
func NewDummyTransaction(packageList ...Package) Transaction {
	return dummyTransaction{packageList}
}

// FakePackageList - Returns a list of count fake packages.
func FakePackageList(count int) []Package {
	var err error
	var fake *faker.Faker
	if fake, err = faker.New("en"); err != nil {
		panic(err)
	}

	packages := make([]Package, 0, count)

	for i := 0; i < count; i++ {
		p := Package{
			ID:      fake.Characters(64),
			Info:    InfoEnum(rand.Uint32()),
			Summary: strings.Join(fake.Words(10, false), " "),
		}
		packages = append(packages, p)
	}

	return packages
}

// dummyClient - Dummy packagekit client for testing purposes.
type dummyClient struct {
	PackageList []Package
}

func (t dummyClient) Shutdown() error {
	return nil
}

// CreateTransaction - Returns a new dummy Transaction for testing purposes.
func (t dummyClient) CreateTransaction() (Transaction, error) {
	return NewDummyTransaction(t.PackageList...), nil
}

// TransactionList - NotImplemented
func (t dummyClient) TransactionList() ([]Transaction, error) {
	return nil, errNotImplemented
}

// CanAuthorize - NotImplemented
func (t dummyClient) CanAuthorize(actionID string) (uint32, error) {
	return 0, errNotImplemented
}

// DaemonState - NotImplemented
func (t dummyClient) DaemonState() (string, error) {
	return "", errNotImplemented
}

// SuggestDaemonQuit - NotImplemented
func (t dummyClient) SuggestDaemonQuit() error {
	return errNotImplemented
}

// dummyTransaction - Dummy packagekit transaction for testing purposes.
type dummyTransaction struct {
	PackageList []Package
}

// Cancel - NotImplemented
func (t dummyTransaction) Cancel() error {
	return nil
}

// Packages - Returns the list of packages stored in the struct.
func (t dummyTransaction) Packages(filter PackageFilter) ([]Package, error) {
	return t.PackageList, nil
}

// InstallPackages - Installs the list of packages.
func (t dummyTransaction) InstallPackages(options TransactionFlag, packageIDs ...string) error {
	return nil
}

// Resolve
func (t dummyTransaction) Resolve(filter PackageFilter, packageIDs ...string) ([]Package, error) {
	return t.PackageList, nil
}

// DownloadPackages - NotImplemented
func (t dummyTransaction) DownloadPackages(storeInCache bool, packageIDs ...string) error {
	return errNotImplemented
}

func (t dummyTransaction) RefreshCache() error {
	return nil
}
