// +build linux

package packagekit

import "github.com/godbus/dbus"

// NewClient -  Returns dbusClient for packagekit.
func NewClient() (Client, error) {
	systemBus, err := dbus.SystemBus()
	if err != nil {
		return nil, err
	}

	return dbusClient{systemBus: systemBus, pkgKit: systemBus.Object(pkDbusInterface, pkDbusObjectPath)}, nil
}
