package deployment

import "bitbucket.org/jatone/bearded-wookie/packagekit"

type Package packagekit.Package
type PackageByID []Package

// Methods required by sort Interface
func (t PackageByID) Len() int           { return len(t) }
func (t PackageByID) Less(i, j int) bool { return t[i].ID < t[j].ID }
func (t PackageByID) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }

type Coordinator interface {
	// Status of the deployment coordinator
	// idle, deploying, locked
	Status() error

	// SystemStateChecksum returns a hash describing the state
	// of the server.
	SystemStateChecksum() ([]byte, error)

	// // Install the manifest for the system.
	// // the manifest is a toml file describing the repositories
	// // and the packages to be installed/updated.
	// InstallManifest([]byte) error
	//
	// // Initiates a repository fetch and installation of the packages specified
	// // within the manifest.
	// Deploy() error
	//
	// // Locks the system preventing deployments from being performed.
	// Lock() error
	// // Unlocks the system allowing deployments to be performed.
	// Unlock() error

	// Installs a list of packages
	//
	// packageIDs - array of package identifiers describing what packages to install.
	// Must be formatted according to
	// http://www.freedesktop.org/software/PackageKit/gtk-doc/concepts.html#introduction-ideas-packageid
	InstallPackages(packageIDs ...string) error
}
