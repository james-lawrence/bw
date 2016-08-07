// +build darwin

package packagekit

// NewClient - returns DummyClient since darwin systems do not have packagekit.
func NewClient() (Client, error) {
	return NewDummyClient(), nil
}
