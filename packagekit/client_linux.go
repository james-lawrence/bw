// +build linux

package packagekit

import "github.com/godbus/dbus"

// NewClient -  Returns dbusClient for packagekit.
func NewClient() (Client, error) {
	systemBus, err := dbus.SystemBus()
	if err != nil {
		return nil, err
	}

	return conn{systemBus: systemBus, pkgKit: systemBus.Object(pkDbusInterface, pkDbusObjectPath)}, nil
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
