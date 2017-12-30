// +build linux

package packagekit_test

import (
	"context"

	. "github.com/james-lawrence/bw/packagekit"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Conn", func() {
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
				allInstalledPackages, err := firstTransaction.Packages(context.Background(), FilterInstalled)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(allInstalledPackages)).To(BeNumerically(">", 0))

				// Get the list of installed devel packages
				secondTransaction, err := client.CreateTransaction()
				Expect(err).ToNot(HaveOccurred())
				installedDevelPackages, err := secondTransaction.Packages(context.Background(), FilterDevel|FilterInstalled)
				Expect(err).ToNot(HaveOccurred())
				// Make sure allInstalledPackages is greater than installedDevelPackages
				Expect(len(installedDevelPackages)).To(BeNumerically(">", 0))
			})
		})

		Describe("Resolve", func() {
			It("Resolve a list of package names", func() {
				transaction, err := client.CreateTransaction()
				Expect(err).ToNot(HaveOccurred())
				pset, err := transaction.Resolve(context.Background(), FilterNone|FilterDevel, "htop")
				Expect(err).ToNot(HaveOccurred())
				Expect(pset).To(HaveLen(1))
			})
		})

		Describe("InstallPackages", func() {
			It("Installs a list of packages successfully", func() {
				// TODO: https://github.com/hughsie/PackageKit/issues/229
				// transaction, err := client.CreateTransaction()
				// Expect(err).ToNot(HaveOccurred())
				// pset := []string{"htop"}
				// ppset, err := transaction.Resolve(context.Background(), FilterNone, pset...)
				// Expect(err).ToNot(HaveOccurred())
				// pset = make([]string, 0, len(ppset))
				// for _, p := range ppset {
				// 	pset = append(pset, p.ID)
				// }
				p := Package{
					ID: "htop;2.0.2-2;x86_64;extra",
				}

				transaction, err := client.CreateTransaction()
				Expect(err).ToNot(HaveOccurred())
				Expect(transaction.InstallPackages(context.Background(), TransactionFlagSimulate, p)).ToNot(HaveOccurred())
			})
		})

		Describe("Cancel", func() {
			// This will fail if we haven't already called another method on
			// the transaction. Pending until we have an efficient way to handle this.
			It("should not return an error", func() {
				transaction, err := client.CreateTransaction()
				Expect(err).ToNot(HaveOccurred())

				started := make(chan struct{})
				// TODO call a method on the transaction.
				go func() {
					close(started)
					transaction.RefreshCache(context.Background())
				}()
				<-started
				err = IgnoreNotSupported(transaction.Cancel())
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Describe("DownloadPackages", func() {
			It("should not return an error", func() {
				transaction, err := client.CreateTransaction()
				Expect(err).ToNot(HaveOccurred())
				_, err = transaction.DownloadPackages(context.Background(), false)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
