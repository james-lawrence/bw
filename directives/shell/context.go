package shell

import (
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// DefaultContext creates the context for the current machine.
func DefaultContext() (ctx Context, err error) {
	var (
		hostname string
		u        *user.User
		_fqdn    string
	)

	if u, err = user.Current(); err != nil {
		return ctx, errors.Wrap(err, "failed to lookup current user")
	}

	if hostname, err = os.Hostname(); err != nil {
		return ctx, errors.Wrap(err, "failed to lookup hostname")
	}

	if _fqdn, err = fqdn(hostname); err != nil {
		return ctx, err
	}

	return Context{
		Shell:     os.Getenv("SHELL"),
		User:      *u,
		Hostname:  hostname,
		FQDN:      _fqdn,
		Domain:    _domain(_fqdn),
		MachineID: machineID(),
		Environ:   os.Environ(),
		timeout:   5 * time.Minute,
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

// OptionDir set working directory for the command
func OptionDir(l string) Option {
	return func(ctx *Context) {
		ctx.dir = l
	}
}

// OptionAppendEnviron append to the environment for shell commands.
func OptionAppendEnviron(l ...string) Option {
	return func(ctx *Context) {
		ctx.Environ = append(ctx.Environ, l...)
	}
}

// OptionLenient mark the context as lenient, allowing commands to fail.
func OptionLenient(ctx *Context) {
	ctx.lenient = true
}

// OptionTimeout set the timeout for the context.
func OptionTimeout(d time.Duration) Option {
	return func(ctx *Context) {
		ctx.timeout = d
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
	dir       string
	timeout   time.Duration
	lenient   bool
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
