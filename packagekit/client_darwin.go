// +build darwin

package packagekit

// NewClient - returns DummyClient since darwin systems do not have packagekit.
func NewClient() (Client, error) {
	return NewDummyClient(), nil
}

// NewTransaction convience method for getting a transaction directly.
func NewTransaction() (c Client, tx Transaction, err error) {
	if c, err = NewClient(); err != nil {
		return nil, nil, err
	}

	if tx, err = c.CreateTransaction(); err != nil {
		c.Shutdown()
		return nil, nil, err
	}

	return c, tx, nil
}
