package packagekit

import "fmt"

//go:generate stringer -type=ErrorEnum -type=InfoEnum -type=ExitEnum

// Generic NotImplemented Error
var errNotImplemented = fmt.Errorf("Not Implemented")

const pkDbusInterface = "org.freedesktop.PackageKit"
const pkTransactionDbusInterface = "org.freedesktop.PackageKit.Transaction"
const pkDbusObjectPath = "/org/freedesktop/PackageKit"

// PackageKit - DBUS api for installing packages.
// Home Page - http://www.freedesktop.org/software/PackageKit
// Concepts - http://www.freedesktop.org/software/PackageKit/gtk-doc/concepts.html
// PackageID Format - http://www.freedesktop.org/software/PackageKit/gtk-doc/concepts.html#introduction-ideas-packageid
// API Reference - http://www.freedesktop.org/software/PackageKit/gtk-doc/api-reference.html
// Git Repository - https://github.com/hughsie/PackageKit

// Client for interacting with the PackageKit API.
//
// Currently Implements a subset of the complete API.
type Client interface {
	// Transaction - create a new transaction
	//
	// org.freedesktop.PackageKit.CreateTransaction
	CreateTransaction() (Transaction, error)

	// TransactionList - returns a list of current transactions.
	//
	// org.freedesktop.PackageKit.GetTransactionList
	TransactionList() ([]Transaction, error)

	// CanAuthorize - Check if client can perform specified actions.
	//
	// org.freedesktop.PackageKit.CanAuthorize
	// results in either: yes, no, or interactive
	CanAuthorize(actionID string) (uint32, error)

	// DaemonState - Queries the daemon for debugging information
	//
	// strictly for reference information, no secure information will be
	// provided.
	// org.freedesktop.PackageKit.GetDaemonState
	DaemonState() (string, error)

	// SuggestDaemonQuit - Suggests to the daemon it should quit asap.
	//
	// org.freedesktop.PackageKit.SuggestDaemonQuit
	SuggestDaemonQuit() error
}

// Transaction - Describing the PackageKit Transaction API.
//
// Currently implements a subset of the transaction API.
type Transaction interface {
	// Cancel - org.freedesktop.PackageKit.Transaction.Cancel
	// Cancel this transaction.
	Cancel() error

	// Packages - org.freedesktop.PackageKit.Transaction.GetPackages
	//
	// Emits all packages matching the specified filter.
	//   err := tx.Packages("none")
	//   err := tx.Packages("installed;~devel")
	Packages(filter PackageFilter) ([]Package, error)

	// Installs a list of packages
	//
	// packageIDs - array of package identifiers describing what packages to install.
	// Must be formatted according to
	// http://www.freedesktop.org/software/PackageKit/gtk-doc/concepts.html#introduction-ideas-packageid
	// example: htop;;;
	// example: htop;2.0.2-1;;
	// example: htop;2.0.2-1;x86_64;
	InstallPackages(packageIDs ...string) error

	// InstallLocalFile(paths ...string) error

	// Download the packages
	//
	// storeInCache - Whether we should store the downloaded packages in the cache.
	//   err := tx.DownloadPackages(true, "gnome-shell")
	// packageIDs - array of package identifiers describing what packages to download.
	DownloadPackages(storeInCache bool, packageIDs ...string) error

	RefreshCache() error
}

const methodDBUSAddMatch = "org.freedesktop.DBus.AddMatch"
const methodDBUSRemoveMatch = "org.freedesktop.DBus.RemoveMatch"

// packagekit dbus methods.
const methodTransactionDownloadPackages = "org.freedesktop.PackageKit.Transaction.DownloadPackages"
const methodTransactionGetPackages = "org.freedesktop.PackageKit.Transaction.GetPackages"
const methodTransactionCancel = "org.freedesktop.PackageKit.Transaction.Cancel"
const methodTransactionRefreshCache = "org.freedesktop.PackageKit.Transaction.RefreshCache"
const methodTransactionInstallPackages = "org.freedesktop.PackageKit.Transaction.InstallPackages"
const methodCreateTransaction = "org.freedesktop.PackageKit.CreateTransaction"

// packagekit dbus signals
const signalTransactionPackage = "org.freedesktop.PackageKit.Transaction.Package"
const signalTransactionFinished = "org.freedesktop.PackageKit.Transaction.Finished"
const signalTransactionError = "org.freedesktop.PackageKit.Transaction.ErrorCode"
const signalTransactionItemProgress = "org.freedesktop.PackageKit.Transaction.ItemProgress"
const signalTransactionStatus = "org.freedesktop.PackageKit.Transaction.Status"
const signalTransactionDestroy = "org.freedesktop.PackageKit.Transaction.Destroy"

func MaybeCancel(tx Transaction, err error) error {
	if err != nil {
		tx.Cancel()
		return err
	}

	return nil
}

// Package Provides basic Information about a package.
type Package struct {
	ID      string
	Info    uint32
	Summary string
}

// PackageFilter Bitwise Filter for searching packages see the constants below for their values.
type PackageFilter uint64

// These constants are calculated from the glib library within the packagekit repository.
// They are calculated by doing a bit shift with their position in the enum list.
// e.g.) FilterUnknown is 1 << 0 which results in 1.
//       FilterNone is 1 << 1 which 2.
//       FilterInstalled is 1 << 2 which is 4. etc, etc.
// source - https://github.com/hughsie/PackageKit/blob/master/lib/packagekit-glib2/pk-enum.h
const (
	FilterUnknown        PackageFilter = 0x0000001
	FilterNone                         = 0x0000002
	FilterInstalled                    = 0x0000004
	FilterNotInstalled                 = 0x0000008
	FilterDevel                        = 0x0000010
	FilterNotDevel                     = 0x0000020
	FilterGui                          = 0x0000040
	FilterNotGui                       = 0x0000080
	FilterFree                         = 0x0000100
	FilterNotFree                      = 0x0000200
	FilterVisible                      = 0x0000400
	FilterNotVisible                   = 0x0000800
	FilterSupported                    = 0x0001000
	FilterNotSupported                 = 0x0002000
	FilterBasename                     = 0x0004000
	FilterNotBasename                  = 0x0008000
	FilterNewest                       = 0x0010000
	FilterNotNewest                    = 0x0020000
	FilterArch                         = 0x0040000
	FilterNotArch                      = 0x0080000
	FilterSource                       = 0x0100000
	FilterNotSource                    = 0x0200000
	FilterCollections                  = 0x0400000
	FilterNotCollections               = 0x0800000
	FilterApplication                  = 0x1000000
	FilterNotApplication               = 0x2000000
)

// TransactionFlag Bitwise enum for use with PackageKit transaction methods.
type TransactionFlag uint64

// These constants are calculated from the glib library within the packagekit repository.
// They are calculated by doing a bit shift with their position in the enum list.
// e.g.) TransactionFlagNone is 1 << 0 which results in 1.
//       TransactionFlagOnlyTrusted is 1 << 1 which 2.
//       TransactionFlagSimulate is 1 << 2 which is 4. etc, etc.
// source - https://github.com/hughsie/PackageKit/blob/master/lib/packagekit-glib2/pk-enum.h
const (
	TransactionFlagNone TransactionFlag = 1 << iota
	TransactionFlagOnlyTrusted
	TransactionFlagSimulate
	TransactionFlagOnlyDownload
	TransactionFlagAllowReinstall
	TransactionFlagJustReinstall
	TransactionFlagAllowDowngrade
)

// ExitEnum type representing exit codes
type ExitEnum uint64

// These constants represent the exit status of commands.
const (
	ExitUnknown ExitEnum = iota
	ExitSuccess
	ExitFailed
	ExitCancelled
	ExitKeyRequired
	ExitEULARequired
	ExitKilled
	ExitMediaChangeRequired
	ExitNeedUntrusted
	ExitCancelledPriority
	ExitSkipTransaction
	ExitRepairRequired
)

// ErrorEnum type representing errors within packagekit.
type ErrorEnum uint64

// These constants represents errors within packagekit.
const (
	ErrorUnknown ErrorEnum = iota
	ErrorOOM
	ErrorNoNetwork
	ErrorNotSupported
	ErrorInternalError
	ErrorGPGFailure
	ErrorPackageIDInvalid
	ErrorPackageNotInstalled
	ErrorPackageNotFound
	ErrorPackageAlreadyInstalled
	ErrorPackageDownloadFailed
	ErrorGroupNotFound
	ErrorGroupListInvalid
	ErrorDepResolutionFailed
	ErrorFilterInvalid
	ErrorCreateThreadFailed
	ErrorTransactionError
	ErrorTransactionCancelled
	ErrorNoCache
	ErrorRepoNotFound
	ErrorCannotRemoveSystemPackage
	ErrorProcessKill
	ErrorFailedInitialization
	ErrorFailedFinalise
	ErrorFailedConfigParsing
	ErrorCannotCancel
	ErrorCannotGetLock
	ErrorNoPackagesToUpdate
	ErrorCannotWriteRepoConfig
	ErrorLocalInstallFailed
	ErrorBadGPGSignature
	ErrorMissingGpgSignature
	ErrorCannotInstallSourcePackage
	ErrorRepoConfigurationError
	ErrorNoLicenseAgreement
	ErrorFileConflicts
	ErrorPackageConflicts
	ErrorRepoNotAvailable
	ErrorInvalidPackageFile
	ErrorPackageInstallBlocked
	ErrorPackageCorrupt
	ErrorAllPackagesAlreadyInstalled
	ErrorFileNotFound
	ErrorNoMoreMirrorsToTry
	ErrorNoDistroUpgradeData
	ErrorIncompatibleArchitecture
	ErrorNoSpaceOnDevice
	ErrorMediaChangeRequired
	ErrorNotAuthorized
	ErrorUpdateNotFound
	ErrorCannotInstallRepoUnsigned
	ErrorCannotUpdateRepoUnsigned
	ErrorCannotGetFilelist
	ErrorCannotGetRequires
	ErrorCannotDisableRepository
	ErrorRestrictedDownload
	ErrorPackageFailedToConfigure
	ErrorPackageFailedToBuild
	ErrorPackageFailedToInstall
	ErrorPackageFailedToRemove
	ErrorUpdateFailedDueToRunningProcess
	ErrorPackageDatabaseChanged
	ErrorProvideTypeNotSupported
	ErrorInstallRootInvalid
	ErrorCannotFetchSources
	ErrorCancelledPriority
	ErrorUnfinishedTransaction
	ErrorLockRequired
	ErrorRepoAlreadySet
)

// InfoEnum The enumerated types used in Package() - these have to refer to a specific
// package action, rather than a general state
type InfoEnum uint64

// The enumerated types used in Package() - these have to refer to a specific
// package action, rather than a general state
const (
	InfoUnknown InfoEnum = iota
	InfoInstalled
	InfoAvailable
	InfoLow
	InfoEnhancement
	InfoNormal
	InfoBugfix
	InfoImportant
	InfoSecurity
	InfoBlocked
	InfoDownloading
	InfoUpdating
	InfoInstalling
	InfoRemoving
	InfoCleanup
	InfoObsoleting
	InfoCollectionInstalled
	InfoCollectionAvailable
	InfoFinished
	InfoReinstalling
	InfoDowngrading
	InfoPreparing
	InfoDecompressing
	InfoUntrusted
	InfoTrusted
	InfoUnavailable
)
