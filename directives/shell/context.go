package shell

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/user"
	"strings"

	"github.com/pkg/errors"
)

// DefaultContext creates the context for the current machine.
func DefaultContext() (ctx Context, err error) {
	var (
		hostname string
		// machineID string
		u    *user.User
		fqdn string
	)

	if u, err = user.Current(); err != nil {
		return ctx, errors.Wrap(err, "failed to lookup current user")
	}

	if hostname, err = os.Hostname(); err != nil {
		return ctx, errors.Wrap(err, "failed to lookup hostname")
	}

	if fqdn, err = _fqdn(hostname); err != nil {
		return ctx, err
	}

	return Context{
		Shell:     os.Getenv("SHELL"),
		User:      *u,
		Hostname:  hostname,
		FQDN:      fqdn,
		Domain:    _domain(fqdn),
		MachineID: _machineID(),
		Environ:   os.Environ(),
		output:    ioutil.Discard,
	}, nil
}

// Option context option.
type Option func(*Context)

// OptionLogger set the logger for the context.
func OptionLogger(l logger) Option {
	return func(ctx *Context) {
		ctx.output = logging{logger: l}
	}
}

// OptionEnviron set the environment for shell commands.
func OptionEnviron(l []string) Option {
	return func(ctx *Context) {
		ctx.Environ = l
	}
}

// NewContext creates a new context using the provided context as a base and then applies options.
func NewContext(tmp Context, options ...Option) Context {
	for _, opt := range options {
		opt(&tmp)
	}

	return tmp
}

// Context ...
type Context struct {
	Shell     string
	User      user.User
	Hostname  string
	MachineID string
	Domain    string
	FQDN      string
	Environ   []string
	output    io.Writer
}

// %H hostname
// %m machine id.
// %d domain name
// %f FQDN
// %u user name
// %U user uid.
// %h user home directory
// %% percent symbol
func (t Context) variableSubst(cmd string) string {
	cmd = strings.Replace(cmd, "%H", t.Hostname, -1)
	cmd = strings.Replace(cmd, "%m", t.MachineID, -1)
	cmd = strings.Replace(cmd, "%d", t.Domain, -1)
	cmd = strings.Replace(cmd, "%f", t.FQDN, -1)
	cmd = strings.Replace(cmd, "%u", t.User.Username, -1)
	cmd = strings.Replace(cmd, "%U", t.User.Uid, -1)
	cmd = strings.Replace(cmd, "%h", t.User.HomeDir, -1)
	cmd = strings.Replace(cmd, "%%", "%", -1)

	return cmd
}

func _domain(fqdn string) string {
	idx := strings.Index(fqdn, ".")
	if idx == -1 {
		return fqdn
	}

	return fqdn[idx:]
}

func _fqdn(hostname string) (string, error) {
	addrs, err := net.LookupIP(hostname)
	if err != nil {
		return "", errors.Wrap(err, "failed to lookup ip for fqdn")
	}

	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			ip, err := ipv4.MarshalText()
			if err != nil {
				return "", errors.Wrap(err, "failed to marshal ip for fqdn")
			}
			hosts, err := net.LookupAddr(string(ip))
			if err != nil {
				return "", errors.Wrap(err, "failed to lookup hosts for addr")
			}

			for _, fqdn := range hosts {
				return strings.TrimSuffix(fqdn, "."), nil // return fqdn without trailing dot
			}
		}
	}

	// no FQDN found
	return "", nil
}

func _machineID() string {
	var (
		err error
		raw []byte
	)

	if raw, err = ioutil.ReadFile("/etc/machine-id"); err != nil {
		log.Println("failed to read machine id, defaulting to empty string", err)
		return ""
	}

	return strings.TrimSpace(string(raw))
}
