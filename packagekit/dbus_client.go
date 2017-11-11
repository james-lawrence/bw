package packagekit

import (
	"fmt"
	"log"
	"time"

	"strings"

	"github.com/godbus/dbus"
	"github.com/pkg/errors"
)

// dbusClient - Client configured to interact with the PackageKit api on a single
// systemBus connection. Implements the Client interface.
type dbusClient struct {
	systemBus *dbus.Conn     // dbus system bus connection.
	pkgKit    dbus.BusObject // packagekit dbus object
}

func (t dbusClient) Shutdown() error {
	return t.systemBus.Close()
}

// CreateTransaction - Creates a PackageKit transaction on a private systemBus
// connection.
func (t dbusClient) CreateTransaction() (Transaction, error) {
	var (
		err           error
		transactionID dbus.ObjectPath
	)

	// get a new systemBus connection, as there can only be one transaction per
	// connection.
	systemBus, err := dbus.SystemBusPrivate()
	if err != nil {
		return dbusTransaction{}, err
	}

	// setup code for private system bus. The godbus package should be handling
	// all this in the invoked method.....
	if err = systemBus.Auth(nil); err != nil {
		systemBus.Close()
		return dbusTransaction{}, err
	}

	if err = systemBus.Hello(); err != nil {
		systemBus.Close()
		return dbusTransaction{}, err
	}

	channel := make(chan *dbus.Signal, 1000)
	systemBus.Signal(channel)

	pkgKitObject := systemBus.Object(pkDbusInterface, pkDbusObjectPath)
	if err = pkgKitObject.Call(methodCreateTransaction, 0).Store(&transactionID); err != nil {
		return dbusTransaction{}, err
	}

	transactionObject := systemBus.Object(pkDbusInterface, transactionID)
	return dbusTransaction{systemBus: systemBus, transaction: transactionObject, signalChan: channel}, nil
}

// TransactionList - Not implemented.
func (t dbusClient) TransactionList() ([]Transaction, error) {
	return []Transaction{}, errNotImplemented
}

// CanAuthorize - Not implemented.
func (t dbusClient) CanAuthorize(actionID string) (uint32, error) {
	return 0, errNotImplemented
}

// DaemonState - Not implemented.
func (t dbusClient) DaemonState() (string, error) {
	return "", errNotImplemented
}

// SuggestDaemonQuit - Not implemented.
func (t dbusClient) SuggestDaemonQuit() error {
	return errNotImplemented
}

// dbusTransaction - Represents a single PackageKit Transaction.
type dbusTransaction struct {
	systemBus   *dbus.Conn        // dbus system bus connection.
	transaction dbus.BusObject    // packagekit transaction dbus object
	signalChan  chan *dbus.Signal // channel for recieving signals
}

// Cancel - Cancel the current Transaction.
func (t dbusTransaction) Cancel() error {
	return t.transaction.Call(methodTransactionCancel, 0).Store()
}

// Packages - Returns a list of Packages filtered according to the provided Filter.
func (t dbusTransaction) Packages(filter PackageFilter) ([]Package, error) {
	var (
		err      error
		pathRule = "path=" + string(t.transaction.Path())
		packages = make([]Package, 0, 1000)
	)

	signals := []string{
		transactionSignal(pathRule, "member=Package"),
		transactionSignal(pathRule, "member=Finished"),
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

		if event, err = awaitEvent(10*time.Second, t.signalChan); err != nil {
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

func (t dbusTransaction) RefreshCache() error {
	var (
		err error
	)
	pathRule := "path=" + string(t.transaction.Path())
	signals := []string{
		transactionSignal(pathRule),
	}

	// Listen for signals
	for _, signal := range signals {
		if err = t.listenFor(signal); err != nil {
			return err
		}
		defer t.stopListeningFor(signal)
	}

	if err = t.transaction.Call(methodTransactionRefreshCache, dbus.Flags(0), true).Store(); err != nil {
		return errors.Wrap(err, "failed to invoked refresh cache")
	}

	// Wait for the method to finish
	duration, err := awaitCompletion(10*time.Second, t.signalChan)
	log.Println("RefreshCache completed in", duration)
	return err
}

// InstallPackages - Installs a list of Packages.
func (t dbusTransaction) InstallPackages(packageIDs ...string) error {
	var (
		err error
	)
	pathRule := "path=" + string(t.transaction.Path())
	signals := []string{
		transactionSignal(pathRule),
	}

	// Listen for signals
	for _, signal := range signals {
		if err = t.listenFor(signal); err != nil {
			return err
		}
		defer t.stopListeningFor(signal)
	}

	flags := dbus.Flags(0)
	transactionFlags := TransactionFlag(TransactionFlagNone | TransactionFlagAllowDowngrade)
	if err = t.transaction.Call(methodTransactionInstallPackages, flags, transactionFlags, packageIDs).Store(); err != nil {
		return errors.Wrap(err, "failed to install packages")
	}

	// Wait for the method to finish
	_, err = awaitCompletion(10*time.Second, t.signalChan)
	return err
}

// DownloadPackages - NotImplemented
func (t dbusTransaction) DownloadPackages(storeInCache bool, packageIDs ...string) (err error) {
	const (
		timeout = 10 * time.Second
	)
	var (
		duration time.Duration
	)

	if len(packageIDs) == 0 {
		return nil
	}

	pathRule := "path=" + string(t.transaction.Path())
	signals := []string{
		transactionSignal(pathRule),
	}

	// Listen for signals
	for _, signal := range signals {
		if err = t.listenFor(signal); err != nil {
			return err
		}
		defer t.stopListeningFor(signal)
	}

	if err = t.transaction.Call(methodTransactionDownloadPackages, dbus.Flags(0), storeInCache, packageIDs).Store(); err != nil {
		return errors.Wrap(err, "failed to download packages")
	}

	// Wait for the method to finish
	duration, err = awaitCompletion(timeout, t.signalChan)
	log.Println("downloadPackages completed in", duration)
	return err
}

func awaitCompletion(timeout time.Duration, c chan *dbus.Signal) (time.Duration, error) {
	start := time.Now()
	d := time.NewTimer(timeout)
	defer d.Stop()
	for {
		select {
		case _ = <-d.C:
			return time.Now().Sub(start), errors.Errorf("operation timed out: %d", timeout)
		case event := <-c:
			d.Reset(timeout)
			switch event.Name {
			case signalTransactionFinished:
				err := handleFinished(event)
				duration := exitDuration(err)
				return duration, ignoreSuccess(err)
			case signalTransactionError:
				return time.Now().Sub(start), handleError(event)
			case signalTransactionDestroy:
				return time.Now().Sub(start), nil
			}
		}
	}
}

func awaitEvent(timeout time.Duration, c chan *dbus.Signal) (*dbus.Signal, error) {
	d := time.NewTimer(timeout)
	defer d.Stop()
	select {
	case _ = <-d.C:
		return nil, errors.Errorf("operation timed out: %d", timeout)
	case event := <-c:
		d.Reset(timeout)
		switch event.Name {
		case signalTransactionFinished:
			return event, ignoreSuccess(handleFinished(event))
		case signalTransactionError:
			return event, handleError(event)
		}
		return event, nil
	}
}

func propertiesSignal(rules ...string) string {
	return fmt.Sprintf("type='signal',interface='%s',%s", "org.freedesktop.DBus.Properties", strings.Join(rules, ","))
}

func transactionSignal(rules ...string) string {
	signal := fmt.Sprintf("type='signal',interface='%s',%s", pkTransactionDbusInterface, strings.Join(rules, ","))
	return signal
}

func (t dbusTransaction) listenFor(signal string) error {
	return t.systemBus.BusObject().Call(methodDBUSAddMatch, 0, signal).Err
}

func (t dbusTransaction) stopListeningFor(signal string) error {
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

func handleError(signal *dbus.Signal) error {
	var (
		err  string
		code uint32
	)

	if decodeErr := errors.Wrapf(dbus.Store(signal.Body, &code, &err), "failed to decode %s", signalTransactionError); decodeErr != nil {
		return decodeErr
	}

	return transactionError{code: ErrorEnum(code), msg: err}
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
