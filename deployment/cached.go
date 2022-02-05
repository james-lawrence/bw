package deployment

// Cached represents a deployment that is simply a caching server.
// it doesn't actually deploy the code, but can act as a source for downloading.
type Cached struct{}

// Deploy ...
func (t Cached) Deploy(dctx *DeployContext) {
	go t.deploy(dctx)
}

func (t Cached) deploy(dctx *DeployContext) {
	dctx.Done(nil)
}
