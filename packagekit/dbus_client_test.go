// +build linux

package packagekit_test

import (
	. "github.com/james-lawrence/bw/packagekit"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DbusClient", func() {
	Describe("Client", func() {
		var client Client

		BeforeEach(func() {
			var err error
			client, err = NewClient()
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("CreateTransaction", func() {
			It("should not return an error", func() {
				// ignoring transaction.
				_, err := client.CreateTransaction()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Describe("TransactionList", func() {
			It("should return errNotImplemented", func() {
				// Ignoring transaction list.
				_, err := client.TransactionList()
				Expect(err).To(MatchError("Not Implemented"))
			})
		})

		Describe("CanAuthorize", func() {
			It("should return errNotImplemented", func() {
				// ignoring resulting boolean value.
				_, err := client.CanAuthorize("")
				Expect(err).To(MatchError("Not Implemented"))
			})
		})

		Describe("DaemonState", func() {
			It("should return errNotImplemented", func() {
				// ignoring daemon state.
				_, err := client.DaemonState()
				Expect(err).To(MatchError("Not Implemented"))
			})
		})

		Describe("SuggestDaemonQuit", func() {
			It("should return errNotImplemented", func() {
				err := client.SuggestDaemonQuit()
				Expect(err).To(MatchError("Not Implemented"))
			})
		})
	})

	Describe("Transaction", func() {
		var client Client

		BeforeEach(func() {
			var err error
			client, err = NewClient()
			Expect(err).ToNot(HaveOccurred())

		})

		AfterEach(func() {
			Expect(client.Shutdown()).ToNot(HaveOccurred())
		})

		Describe("Packages", func() {
			It("returns a list of packages, filtered with the provided filter", func() {
				// Get the list of all installed packages and verify that it has a length > 0
				firstTransaction, err := client.CreateTransaction()
				Expect(err).ToNot(HaveOccurred())
				allInstalledPackages, err := firstTransaction.Packages(FilterInstalled)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(allInstalledPackages)).To(BeNumerically(">", 0))

				// Get the list of installed devel packages
				secondTransaction, err := client.CreateTransaction()
				Expect(err).ToNot(HaveOccurred())
				installedDevelPackages, err := secondTransaction.Packages(FilterDevel | FilterInstalled)
				Expect(err).ToNot(HaveOccurred())
				// Make sure allInstalledPackages is greater than installedDevelPackages
				Expect(len(installedDevelPackages)).To(BeNumerically(">", 0))
			})
		})

		Describe("InstallPackages", func() {
			PIt("Installs a list of packages successfully", func() {
				transaction, err := client.CreateTransaction()
				Expect(err).ToNot(HaveOccurred())

				packageIDs := []string{"htop;;;"}
				err = transaction.InstallPackages(0, packageIDs...)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Describe("Cancel", func() {
			// This will fail if we haven't already called another method on
			// the transaction. Pending until we have an efficient way to handle this.
			PIt("should not return an error", func() {
				transaction, err := client.CreateTransaction()
				Expect(err).ToNot(HaveOccurred())

				// TODO call a method on the transaction.

				err = transaction.Cancel()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Describe("DownloadPackages", func() {
			It("should not return an error", func() {
				transaction, err := client.CreateTransaction()
				Expect(err).ToNot(HaveOccurred())
				Expect(transaction.DownloadPackages(false)).ToNot(HaveOccurred())
			})
		})
	})
})
