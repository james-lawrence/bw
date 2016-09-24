package adapters

import (
	"net/rpc"

	"bitbucket.org/jatone/bearded-wookie/deployment"
)

// Deployment - RPC implementation of the deployment coordinator
type Deployment struct {
	deployment.Coordinator
}

// Status of the
func (t Deployment) Status(noargs struct{}, noresult *error) (err error) {
	*noresult = t.Coordinator.Status()
	return nil
}

// Deploy trigger a deploy.
func (t Deployment) Deploy(noargs struct{}, noresult *struct{}) error {
	return t.Coordinator.Deploy()
}

// DeploymentClient - RPC client implementation of the deployment coordinator.
type DeploymentClient struct {
	*rpc.Client
}

// Status - ...
func (t DeploymentClient) Status() error {
	var status error
	if err := t.Client.Call("Deployment.Status", struct{}{}, &status); err != nil {
		return err
	}
	return status
}

// Deploy ...
func (t DeploymentClient) Deploy() error {
	return t.Client.Call("Deployment.Deploy", struct{}{}, &struct{}{})
}
