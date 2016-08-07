package adapters

import "net/rpc"

import "bitbucket.org/jatone/bearded-wookie/deployment"

// Deployment - RPC implementation of the deployment coordinator
type Deployment struct {
	deployment.Coordinator
}

// Status of the
func (t Deployment) Status(noargs struct{}, noresult *struct{}) (err error) {
	return t.Coordinator.Status()
}

// SystemStateChecksum - Deployment State of the server.
// returns a []byte (generally a md5sum) used to distinguish servers
// that have different installed software.
func (t Deployment) SystemStateChecksum(args struct{}, state *[]byte) (err error) {
	*state, err = t.Coordinator.SystemStateChecksum()
	return
}

// InstallPackages - installs a list of packages.
func (t Deployment) InstallPackages(packageIDs []string, noresult *struct{}) (err error) {
	return t.Coordinator.InstallPackages(packageIDs...)
}

// DeploymentClient - RPC client implementation of the deployment coordinator.
type DeploymentClient struct {
	*rpc.Client
}

// Status - ...
func (t DeploymentClient) Status() error {
	return t.Client.Call("Deployment.Status", struct{}{}, nil)
}

// SystemStateChecksum - retrieves the system state for the server of the rpc.Client
func (t DeploymentClient) SystemStateChecksum() ([]byte, error) {
	var (
		status []byte
		err    error
	)
	err = t.Client.Call("Deployment.SystemStateChecksum", struct{}{}, &status)
	return status, err
}

// Packages - array of installed packages on the server.
func (t DeploymentClient) Packages() ([]deployment.Package, error) {
	var (
		packages []deployment.Package
		err      error
	)
	err = t.Client.Call("Deployment.Packages", struct{}{}, &packages)
	return packages, err
}

// InstallPackages - installs a list of packages.
func (t DeploymentClient) InstallPackages(packageIDs ...string) error {
	return t.Client.Call("Deployment.InstallPackages", packageIDs, nil)
}
