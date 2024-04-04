package shell

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// DefaultContext creates the context for the current machine.
func DefaultContext(options ...Option) (ctx Context, err error) {
	var (
		dir      string
		hostname string
		u        *user.User
		_fqdn    string
	)

	if dir, err = os.Getwd(); err != nil {
		return ctx, errors.Wrap(err, "failed to lookup current directory")
	}

	if u, err = user.Current(); err != nil {
		return ctx, errors.Wrap(err, "failed to lookup current user")
	}

	if hostname, err = os.Hostname(); err != nil {
		return ctx, errors.Wrap(err, "failed to lookup hostname")
	}

	if _fqdn, err = fqdn(hostname); err != nil {
		return ctx, err
	}

	return NewContext(Context{
		deploymentID:  "",
		WorkDirectory: dir,
		Shell:         os.Getenv("SHELL"),
		User:          *u,
		Hostname:      hostname,
		FQDN:          _fqdn,
		Domain:        _domain(_fqdn),
		MachineID:     machineID(),
		Environ:       os.Environ(),
		timeout:       5 * time.Minute,
		output:        io.Discard,
	}, options...), nil
}

// Option context option.
type Option func(*Context)

// OptionLogger set the logger for the context.
func OptionLogger(l logger) Option {
	return func(ctx *Context) {
		ctx.output = newLogging(l)
	}
}

// OptionEnviron set the environment for shell commands.
func OptionEnviron(l []string) Option {
	return func(ctx *Context) {
		ctx.Environ = l
	}
}

// OptionDir set archive directory for the command
func OptionDir(l string) Option {
	return func(ctx *Context) {
		ctx.dir = l
	}
}

// OptionVCSCommit set vcs commit for the command.
func OptionVCSCommit(l string) Option {
	return func(ctx *Context) {
		ctx.commit = l
	}
}

// OptionWorkDir set work directory
func OptionWorkDir(l string) Option {
	return func(ctx *Context) {
		ctx.WorkDirectory = l
	}
}

// shell.OptionVCSCommit(dctx.Archive.Commit),
// OptionTempDir set temp directory created for bw.
func OptionTempDir(l string) Option {
	return func(ctx *Context) {
		ctx.tmpdir = l
	}
}

// OptionCacheDir set cache directory created for bw.
func OptionCacheDir(l string) Option {
	return func(ctx *Context) {
		ctx.cachedir = l
	}
}

// OptionBWConfigDir set the directory where the bw configuration is located.
// only available locally.
func OptionBWConfigDir(l string) Option {
	return func(ctx *Context) {
		ctx.bwconfigdir = l
	}
}

// OptionAppendEnviron append to the environment for shell commands.
func OptionAppendEnviron(l ...string) Option {
	return func(ctx *Context) {
		ctx.Environ = append(ctx.Environ, l...)
	}
}

// OptionLoadEnv env files to load.
func OptionLoadEnv(l ...string) Option {
	return func(ctx *Context) {
		ctx.loadenv = append(ctx.loadenv, l...)
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

// OptionDeployID the id of the current deployment
func OptionDeployID(id string) Option {
	return func(ctx *Context) {
		ctx.deploymentID = id
	}
}

// OptionDeployID the id of the current deployment
func OptionDeployerDisplayName(name string) Option {
	return func(ctx *Context) {
		ctx.User.Username = name
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
	Shell         string
	User          user.User
	Hostname      string
	MachineID     string
	Domain        string
	FQDN          string
	WorkDirectory string
	commit        string
	Environ       []string
	loadenv       []string
	output        io.Writer
	deploymentID  string
	dir           string
	tmpdir        string
	cachedir      string
	bwconfigdir   string
	timeout       time.Duration
	lenient       bool
}

func (t Context) variableSubst(cmd string) string {
	const escaped = "__BW_ESC__"
	cmd = strings.Replace(cmd, "%%", escaped, -1)
	cmd = strings.Replace(cmd, "%H", t.Hostname, -1)
	cmd = strings.Replace(cmd, "%bw.machine.hostname%", t.Hostname, -1)
	cmd = strings.Replace(cmd, "%m", t.MachineID, -1)
	cmd = strings.Replace(cmd, "%bw.machine.id%", t.MachineID, -1)
	cmd = strings.Replace(cmd, "%d", t.Domain, -1)
	cmd = strings.Replace(cmd, "%bw.machine.domain%", t.Domain, -1)
	cmd = strings.Replace(cmd, "%f", t.FQDN, -1)
	cmd = strings.Replace(cmd, "%bw.machine.fqdn%", t.FQDN, -1)
	cmd = strings.Replace(cmd, "%u", t.User.Username, -1)
	cmd = strings.Replace(cmd, "%bw.user.name%", t.User.Username, -1)
	cmd = strings.Replace(cmd, "%U", t.User.Uid, -1)
	cmd = strings.Replace(cmd, "%bw.user.id%", t.User.Uid, -1)
	cmd = strings.Replace(cmd, "%h", t.User.HomeDir, -1)
	cmd = strings.Replace(cmd, "%bw.user.home.directory%", t.User.HomeDir, -1)
	cmd = strings.Replace(cmd, "%bwroot", t.dir, -1)
	cmd = strings.Replace(cmd, "%bw.archive.directory%", t.dir, -1)
	cmd = strings.Replace(cmd, "%bwtmp", t.tmpdir, -1)
	cmd = strings.Replace(cmd, "%bw.temp.directory%", t.tmpdir, -1)
	cmd = strings.Replace(cmd, "%bwcwd", t.WorkDirectory, -1)
	cmd = strings.Replace(cmd, "%bw.work.directory%", t.WorkDirectory, -1)
	cmd = strings.Replace(cmd, "%bw.cache.directory%", t.cachedir, -1)
	cmd = strings.Replace(cmd, "%bw.config.directory%", t.bwconfigdir, -1)
	cmd = strings.Replace(cmd, "%bw.deploy.id%", t.deploymentID, -1)
	cmd = strings.Replace(cmd, "%bw.deploy.commit%", t.commit, -1)
	cmd = strings.Replace(cmd, escaped, "%", -1)

	return cmd
}

func (t Context) environmentSubst() []string {
	return append(
		t.Environ,
		fmt.Sprintf("BW_ENVIRONMENT_DEPLOY_ID=%s", t.deploymentID),
		fmt.Sprintf("BW_ENVIRONMENT_DEPLOY_COMMIT=%s", t.commit),
		fmt.Sprintf("BW_ENVIRONMENT_HOST=%s", t.Hostname),
		fmt.Sprintf("BW_ENVIRONMENT_MACHINE_ID=%s", t.MachineID),
		fmt.Sprintf("BW_ENVIRONMENT_DOMAIN=%s", t.Domain),
		fmt.Sprintf("BW_ENVIRONMENT_FQDN=%s", t.FQDN),
		fmt.Sprintf("BW_ENVIRONMENT_USERNAME=%s", t.User.Username),
		fmt.Sprintf("BW_ENVIRONMENT_USERID=%s", t.User.Uid),
		fmt.Sprintf("BW_ENVIRONMENT_USERHOME=%s", t.User.HomeDir),
		fmt.Sprintf("BW_ENVIRONMENT_ROOT=%s", t.dir),
		fmt.Sprintf("BW_ENVIRONMENT_ARCHIVE_DIRECTORY=%s", t.dir),
		fmt.Sprintf("BW_ENVIRONMENT_WORK_DIRECTORY=%s", t.WorkDirectory),
		fmt.Sprintf("BW_ENVIRONMENT_TEMP_DIRECTORY=%s", t.tmpdir),
		fmt.Sprintf("BW_ENVIRONMENT_CACHE_DIRECTORY=%s", t.cachedir),
		fmt.Sprintf("BW_ENVIRONMENT_CONFIG_DIRECTORY=%s", t.bwconfigdir),
	)
}

func _domain(fqdn string) string {
	idx := strings.Index(fqdn, ".")
	if idx == -1 {
		return fqdn
	}

	return fqdn[idx:]
}
