package packagekit

import (
	"fmt"
	"log"

	"github.com/godbus/dbus"
)

import "strings"

// dbusClient - Client configured to interact with the PackageKit api on a single
// systemBus connection. Implements the Client interface.
type dbusClient struct {
	systemBus *dbus.Conn     // dbus system bus connection.
	pkgKit    dbus.BusObject // packagekit dbus object
}

// CreateTransaction - Creates a PackageKit transaction on a private systemBus
// connection.
func (t dbusClient) CreateTransaction() (Transaction, error) {
	var err error
	var transactionID dbus.ObjectPath

	// Get a new systemBus connection, as there can only be one transaction per
	// connection.
	systemBus, err := dbus.SystemBusPrivate()
	if err != nil {
		return dbusTransaction{}, err
	}

	// Setup code for private system bus. The godbus package should be handling
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
	if err = pkgKitObject.Call("org.freedesktop.PackageKit.CreateTransaction", 0).Store(&transactionID); err != nil {
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
	return t.transaction.Call("org.freedesktop.PackageKit.Transaction.Cancel", 0).Store()
}

// Packages - Returns a list of Packages filtered according to the provided Filter.
func (t dbusTransaction) Packages(filter PackageFilter) ([]Package, error) {
	var err error
	pathRule := "path=" + string(t.transaction.Path())
	packageSignal := transactionSignal(pathRule, "member=Package")
	finishedSignal := transactionSignal(pathRule, "member=Finished")
	packages := make([]Package, 0, 1000)

	if err = t.systemBus.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, packageSignal).Err; err != nil {
		return nil, err
	}
	defer t.systemBus.BusObject().Call("org.freedesktop.DBus.RemoveMatch", 0, packageSignal)

	if err = t.systemBus.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, finishedSignal).Err; err != nil {
		return nil, err
	}
	defer t.systemBus.BusObject().Call("org.freedesktop.DBus.RemoveMatch", 0, finishedSignal)

	if err = t.transaction.Call("org.freedesktop.PackageKit.Transaction.GetPackages", 0, filter).Store(); err != nil {
		return nil, err
	}

	for event := range t.signalChan {
		switch event.Name {
		case "org.freedesktop.PackageKit.Transaction.Package":
			pkg := Package{}
			dbus.Store(event.Body, &pkg.Info, &pkg.ID, &pkg.Summary)
			packages = append(packages, pkg)
		case signalTransactionFinished:
			return packages, handleFinished(event)
		default:
			// Should never happen should error this out.
			fmt.Printf("event: %#v\n", event.Name)
		}
	}
	panic("should never happen")
}

func (t dbusTransaction) RefreshCache() error {
	// dbus method name
	const methodName = "org.freedesktop.PackageKit.Transaction.RefreshCache"
	var (
		err error
	)
	pathRule := "path=" + string(t.transaction.Path())
	signals := []string{
		// propertiesSignal(pathRule, "member=PropertiesChanged"),
		transactionSignal(pathRule),
		// transactionSignal(pathRule, "member=Progress"),
		// transactionSignal(pathRule, "member=Files"),
		// transactionSignal(pathRule, "member=Status"),
		transactionSignal(pathRule, "member=ErrorCode"),
		transactionSignal(pathRule, "member=Finished"),
	}

	// Listen for signals
	for _, signal := range signals {
		if err = t.listenFor(signal); err != nil {
			return err
		}
		defer t.stopListeningFor(signal)
	}

	flags := dbus.Flags(0)
	if err = t.transaction.Call(methodName, flags, false).Store(); err != nil {
		log.Println("TX failed", err)
		return err
	}

	// Wait for the method to finish
	for event := range t.signalChan {
		switch event.Name {
		case "org.freedesktop.PackageKit.Transaction.ItemProgress":
			log.Println("ItemProgress Signal", event)
		case signalTransactionError:
			log.Println("Error", event)
		case signalTransactionFinished:
			return handleFinished(event)

		default:
			fmt.Println("Default: ")
			fmt.Printf("refresh cache event: %#v\n", event)
		}
	}

	panic("should never happen")
}

// InstallPackages - Installs a list of Packages.
func (t dbusTransaction) InstallPackages(packageIDs ...string) error {
	// dbus method name
	const methodName = "org.freedesktop.PackageKit.Transaction.InstallPackages"
	var (
		err error
	)
	pathRule := "path=" + string(t.transaction.Path())
	signals := []string{
		transactionSignal(pathRule, "member=Progress"),
		transactionSignal(pathRule, "member=Package"),
		transactionSignal(pathRule, "member=Status"),
		transactionSignal(pathRule, "member=ErrorCode"),
		transactionSignal(pathRule, "member=Finished"),
	}

	// Listen for signals
	for _, signal := range signals {
		if err = t.listenFor(signal); err != nil {
			return err
		}
		defer t.stopListeningFor(signal)
	}

	flags := dbus.Flags(0)
	transactionFlags := TransactionFlag(TransactionFlagSimulate)
	if err = t.transaction.Call(methodName, flags, transactionFlags, packageIDs).Store(); err != nil {
		log.Println("TX failed", err)
		return err
	}

	// Wait for the method to finish
	for event := range t.signalChan {
		switch event.Name {
		case "org.freedesktop.PackageKit.Transaction.Progress":
			fmt.Println("Progress signal: ")
			fmt.Printf("%#v\n", event)
		case "org.freedesktop.PackageKit.Transaction.Package":
			fmt.Println("Package signal: ")
			fmt.Printf("%#v\n", event)
		case "org.freedesktop.PackageKit.Transaction.Status":
			fmt.Println("Status signal: ")
			fmt.Printf("%#v\n", event)
		case signalTransactionError:
			var (
				err  string
				code uint32
			)
			dbus.Store(event.Body, &code, &err)
			fmt.Println("Error signal: ", code, err)
			return fmt.Errorf(err)
		case signalTransactionFinished:
			return handleFinished(event)
		default:
			fmt.Println("Default: ")
			fmt.Printf("event: %#v\n", event.Name)
		}
	}
	return nil
}

// DownloadPackages - NotImplemented
func (t dbusTransaction) DownloadPackages(storeInCache bool, packageIDs ...string) error {
	// dbus method name
	const methodName = "org.freedesktop.PackageKit.Transaction.DownloadPackages"
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
	if err = t.transaction.Call(methodName, flags, storeInCache, packageIDs).Store(); err != nil {
		log.Println("TX failed", err)
		return err
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
	return t.systemBus.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, signal).Err
}

func (t dbusTransaction) stopListeningFor(signal string) error {
	return t.systemBus.BusObject().Call("org.freedesktop.DBus.RemoveMatch", 0, signal).Err
}

func handleFinished(signal *dbus.Signal) error {
	var (
		err                 error
		exitstatus, runtime uint32
	)
	fmt.Println("Finished signal: ")
	fmt.Printf("%T, %T\n", signal.Body[0], signal.Body[1])
	if err = dbus.Store(signal.Body, &exitstatus, &runtime); err != nil {
		return err
	}

	if exitstatus != ExitSuccess {
		return fmt.Errorf("received failure exit code: %d", exitstatus)
	}
	log.Printf("install completed in %d milliseconds\n", int(runtime))
	return nil
}
