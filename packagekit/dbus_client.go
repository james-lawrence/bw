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

	for event := range t.signalChan {
		switch event.Name {
		case signalTransactionPackage:
			pkg := Package{}
			dbus.Store(event.Body, &pkg.Info, &pkg.ID, &pkg.Summary)
			packages = append(packages, pkg)
		case signalTransactionFinished:
			duration, err := handleFinished(event)
			log.Println("Packages completed in", duration)
			return packages, err
		default:
			log.Printf("event: %#v\n", event.Name)
		}
	}
	panic("should never happen")
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
	for event := range t.signalChan {
		switch event.Name {
		case signalTransactionItemProgress:
			id, status, progress := handleItemProgress(event)
			log.Println("item progress", id, status, progress)
		case signalTransactionError:
			return handleError(event)
		case signalTransactionFinished:
			duration, err := handleFinished(event)
			log.Println("RefreshCache completed in", duration)
			return err
		default:
			log.Println("Default: ")
			log.Printf("event: %#v\n", event.Name)
		}
	}

	panic("should never happen")
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
	transactionFlags := TransactionFlag(TransactionFlagNone)
	if err = t.transaction.Call(methodTransactionInstallPackages, flags, transactionFlags, packageIDs).Store(); err != nil {
		return errors.Wrap(err, "failed to install packages")
	}

	// Wait for the method to finish
	for event := range t.signalChan {
		switch event.Name {
		case signalTransactionPackage:
			info, pkg, summary := handlePackage(event)
			log.Println("Package:", info, pkg, summary)
		case signalTransactionItemProgress:
			id, status, progress := handleItemProgress(event)
			log.Println("item progress", id, status, progress)
		case signalTransactionError:
			return handleError(event)
		case signalTransactionFinished:
			duration, err := handleFinished(event)
			log.Println("InstallPackages completed in", duration)
			return err
		case signalTransactionDestroy:
			return nil
		}
	}

	panic("should never happen")
}

// DownloadPackages - NotImplemented
func (t dbusTransaction) DownloadPackages(storeInCache bool, packageIDs ...string) error {
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

	if err = t.transaction.Call(methodTransactionDownloadPackages, dbus.Flags(0), storeInCache, packageIDs).Store(); err != nil {
		return errors.Wrap(err, "failed to download packages")
	}

	// Wait for the method to finish
	for event := range t.signalChan {
		switch event.Name {
		default:
			fmt.Println("Default: ")
			fmt.Printf("event: %#v\n", event.Name)
		}
	}
	return nil
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
		log.Println(decodeErr)
	}

	return fmt.Errorf("%s: %s", ErrorEnum(code), err)
}

func handleItemProgress(signal *dbus.Signal) (id string, status, percentage uint32) {
	if err := errors.Wrap(dbus.Store(signal.Body, &id, &status, &percentage), "failed to decode ItemProgress"); err != nil {
		log.Println(err)
	}

	return
}

func handleFinished(signal *dbus.Signal) (time.Duration, error) {
	var (
		err                 error
		duration            time.Duration
		exit                ExitEnum
		exitstatus, runtime uint32
	)

	if err = errors.Wrap(dbus.Store(signal.Body, &exitstatus, &runtime), "failed to decode Finished"); err != nil {
		return 0, err
	}

	exit = ExitEnum(exitstatus)
	duration = time.Duration(runtime) * time.Millisecond

	if exit != ExitSuccess {
		return duration, fmt.Errorf("received failure exit code: %s", exit)
	}

	return duration, nil
}
