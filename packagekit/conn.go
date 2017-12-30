package packagekit

import (
	"context"
	"log"
	"time"

	"github.com/godbus/dbus"
	"github.com/pkg/errors"
)

// conn - Client configured to interact with the PackageKit api on a single
// systemBus connection. Implements the Client interface.
type conn struct {
	systemBus *dbus.Conn     // dbus system bus connection.
	pkgKit    dbus.BusObject // packagekit dbus object
}

func (t conn) Shutdown() error {
	return t.systemBus.Close()
}

// CreateTransaction - Creates a PackageKit transaction on a private systemBus
// connection.
func (t conn) CreateTransaction() (Transaction, error) {
	var (
		err           error
		transactionID dbus.ObjectPath
	)

	// get a new systemBus connection, as there can only be one transaction per
	// connection.
	systemBus, err := dbus.SystemBusPrivate()
	if err != nil {
		return transactionConn{}, err
	}

	// setup code for private system bus. The godbus package should be handling
	// all this in the invoked method.....
	if err = systemBus.Auth(nil); err != nil {
		systemBus.Close()
		return transactionConn{}, err
	}

	if err = systemBus.Hello(); err != nil {
		systemBus.Close()
		return transactionConn{}, err
	}

	channel := make(chan *dbus.Signal, 1000)
	systemBus.Signal(channel)

	pkgKitObject := systemBus.Object(pkDbusInterface, pkDbusObjectPath)
	if err = pkgKitObject.Call(methodCreateTransaction, 0).Store(&transactionID); err != nil {
		return transactionConn{}, err
	}

	transactionObject := systemBus.Object(pkDbusInterface, transactionID)
	return transactionConn{systemBus: systemBus, transaction: transactionObject, signalChan: channel}, nil
}

// TransactionList - Not implemented.
func (t conn) TransactionList() ([]Transaction, error) {
	return []Transaction{}, errNotImplemented
}

// CanAuthorize - Not implemented.
func (t conn) CanAuthorize(actionID string) (uint32, error) {
	return 0, errNotImplemented
}

// DaemonState - Not implemented.
func (t conn) DaemonState() (string, error) {
	return "", errNotImplemented
}

// SuggestDaemonQuit - Not implemented.
func (t conn) SuggestDaemonQuit() error {
	return errNotImplemented
}

// transactionConn - Represents a single PackageKit Transaction.
type transactionConn struct {
	systemBus   *dbus.Conn        // dbus system bus connection.
	transaction dbus.BusObject    // packagekit transaction dbus object
	signalChan  chan *dbus.Signal // channel for recieving signals
}

// Cancel - Cancel the current Transaction.
func (t transactionConn) Cancel() error {
	err := parseDBusError(t.transaction.Call(methodTransactionCancel, 0).Store())
	return errors.WithStack(err)
}

// Packages - Returns a list of Packages filtered according to the provided Filter.
func (t transactionConn) Packages(ctx context.Context, filter PackageFilter) ([]Package, error) {
	var (
		err      error
		pathRule = "path=" + string(t.transaction.Path())
		packages = make([]Package, 0, 100)
	)

	signals := []string{
		transactionSignal(pathRule),
		propertiesSignal(pathRule),
	}

	// Listen for signals
	for _, signal := range signals {
		if err = t.listenFor(signal); err != nil {
			return nil, err
		}
		defer t.stopListeningFor(signal)
	}

	if err = t.transaction.Call(methodTransactionGetPackages, 0, filter).Store(); err != nil {
		return nil, err
	}

	for {
		var (
			event *dbus.Signal
		)

		if event, err = awaitEvent(ctx, t.signalChan); err != nil {
			return packages, err
		}

		switch event.Name {
		case signalTransactionFinished, signalTransactionDestroy:
			return packages, nil
		case signalTransactionPackage:
			pkg := Package{}
			dbus.Store(event.Body, &pkg.Info, &pkg.ID, &pkg.Summary)
			packages = append(packages, pkg)
		}
	}
}

func (t transactionConn) RefreshCache(ctx context.Context) (d time.Duration, err error) {
	pathRule := "path=" + string(t.transaction.Path())
	signals := []string{
		transactionSignal(pathRule),
		propertiesSignal(pathRule),
	}

	// Listen for signals
	for _, signal := range signals {
		if err = t.listenFor(signal); err != nil {
			return d, err
		}
		defer t.stopListeningFor(signal)
	}

	if err = t.transaction.Call(methodTransactionRefreshCache, dbus.Flags(0), true).Store(); err != nil {
		return d, errors.Wrap(err, "failed to invoked refresh cache")
	}

	// Wait for the method to finish
	return awaitCompletion(ctx, t.signalChan)
}

// Resolve - resolves packages into their ids
func (t transactionConn) Resolve(ctx context.Context, filter PackageFilter, packageIDs ...string) (packages []Package, err error) {
	pathRule := "path=" + string(t.transaction.Path())
	signals := []string{
		transactionSignal(pathRule),
		propertiesSignal(pathRule),
	}

	// Listen for signals
	for _, signal := range signals {
		if err = t.listenFor(signal); err != nil {
			return packages, err
		}
		defer t.stopListeningFor(signal)
	}

	flags := dbus.Flags(0)
	if err = t.transaction.Call(methodTransactionResolve, flags, filter, packageIDs).Store(); err != nil {
		return packages, errors.Wrap(err, "failed to resolve packages")
	}

	packages = make([]Package, 0, len(packageIDs))
	for {
		var (
			event *dbus.Signal
		)

		if event, err = awaitEvent(ctx, t.signalChan); err != nil {
			return packages, err
		}

		switch event.Name {
		case signalTransactionPackage:
			info, id, summary := handlePackage(event)
			packages = append(packages, Package{
				ID:      id,
				Info:    info,
				Summary: summary,
			})
		case signalTransactionFinished:
			return packages, err
		}
	}
}

// InstallPackages - Installs a list of Packages.
func (t transactionConn) InstallPackages(ctx context.Context, options TransactionFlag, pset ...Package) error {
	var (
		err error
	)
	pathRule := "path=" + string(t.transaction.Path())
	signals := []string{
		transactionSignal(pathRule),
		propertiesSignal(pathRule),
	}

	// Listen for signals
	for _, signal := range signals {
		if err = t.listenFor(signal); err != nil {
			return err
		}
		defer t.stopListeningFor(signal)
	}

	flags := dbus.Flags(0)
	if err = t.transaction.Call(methodTransactionInstallPackages, flags, options, mapPackageID(pset...)).Store(); err != nil {
		return errors.Wrap(err, "failed to install packages")
	}

	// Wait for the method to finish
	_, err = awaitCompletion(ctx, t.signalChan)
	return err
}

// DownloadPackages - NotImplemented
func (t transactionConn) DownloadPackages(ctx context.Context, storeInCache bool, pset ...Package) (d time.Duration, err error) {
	if len(pset) == 0 {
		return d, nil
	}

	pathRule := "path=" + string(t.transaction.Path())
	signals := []string{
		transactionSignal(pathRule),
	}

	// Listen for signals
	for _, signal := range signals {
		if err = t.listenFor(signal); err != nil {
			return d, err
		}
		defer t.stopListeningFor(signal)
	}

	if err = t.transaction.Call(methodTransactionDownloadPackages, dbus.Flags(0), storeInCache, mapPackageID(pset...)).Store(); err != nil {
		return d, errors.Wrap(err, "failed to download packages")
	}

	// Wait for the method to finish
	return awaitCompletion(ctx, t.signalChan)
}

func (t transactionConn) listenFor(signal string) error {
	return t.systemBus.BusObject().Call(methodDBUSAddMatch, 0, signal).Err
}

func (t transactionConn) stopListeningFor(signal string) error {
	return t.systemBus.BusObject().Call(methodDBUSRemoveMatch, 0, signal).Err
}

func handlePackage(signal *dbus.Signal) (infox InfoEnum, pkg string, summary string) {
	var (
		info uint32
	)

	if err := errors.Wrap(dbus.Store(signal.Body, &info, &pkg, &summary), "failed to decode Package"); err != nil {
		log.Println(err)
	}

	return InfoEnum(info), pkg, summary
}

func handleItemProgress(signal *dbus.Signal) (id string, status, percentage uint32) {
	if err := errors.Wrap(dbus.Store(signal.Body, &id, &status, &percentage), "failed to decode ItemProgress"); err != nil {
		log.Println(err)
	}

	return
}

func exitDuration(err error) time.Duration {
	if exit, ok := err.(exitError); ok {
		return exit.duration
	}

	return 0
}

func ignoreSuccess(err error) error {
	if exit, ok := err.(exitError); ok && exit.code == ExitSuccess {
		return nil
	}

	return err
}

func handleFinished(signal *dbus.Signal) error {
	var (
		err                 error
		exitstatus, runtime uint32
	)

	if err = errors.Wrap(dbus.Store(signal.Body, &exitstatus, &runtime), "decode failure"); err != nil {
		return err
	}

	return exitError{code: ExitEnum(exitstatus), duration: time.Duration(runtime) * time.Millisecond}
}
