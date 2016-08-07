package deployment_test

import (
	. "bitbucket.org/jatone/bearded-wookie/deployment"
	"bitbucket.org/jatone/bearded-wookie/packagekit"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"crypto/md5"
	"encoding/json"
	"hash"
	"sort"
)

var _ = Describe("Coordinator", func() {
	var coordinator Coordinator
	var expectedPackages []Package

	BeforeEach(func() {
		rawPackages := packagekit.FakePackageList(5)
		coordinator = New(packagekit.NewDummyClient(rawPackages...))

		// Convert, then sort the original packageList so that we can compare with result
		expectedPackages = make([]Package, 0, len(rawPackages))
		for _, pkg := range rawPackages {
			expectedPackages = append(expectedPackages, Package(pkg))
		}
		sort.Sort(PackageByID(expectedPackages))
	})

	Describe("Status", func() {
		It("returns nil", func() {
			Expect(coordinator.Status()).To(BeNil())
		})
	})

	Describe("SystemStateChecksum", func() {
		It("returns a checksum of the installed packages on the system", func() {
			result, err := coordinator.SystemStateChecksum()
			Expect(err).ToNot(HaveOccurred())

			// Compute checksum of expectedPackages to compare with result
			var hasher hash.Hash = md5.New()
			json.NewEncoder(hasher).Encode(expectedPackages)
			Expect(result).To(Equal(hasher.Sum(nil)))
		})
	})
})
