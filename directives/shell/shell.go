// Package shell provides the ability to execute shell commands by the agent.
// it supports environment variable substitution, along with a few well known pieces
// of information extracted from the OS.
// Substitutions:
// %H hostname
// %m machine id.
// %d domain name
// %f FQDN
// %u user name
// %U user uid.
// %h user home directory
// %% percent symbol
package shell

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/stringsx"
	"github.com/james-lawrence/bw/internal/timex"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Exec ...
type Exec struct {
	Command string
	Lenient bool
	Retries int16
	Timeout time.Duration
	Environ string
	WorkDir string   `yaml:"directory"`
	LoadEnv []string `yaml:"loadenv"`
}

func (t Exec) execute(ctx context.Context, sctx Context) error {
	timeout := timex.DurationOrDefault(t.Timeout, sctx.timeout)
	deadline, done := context.WithTimeout(ctx, timeout)
	defer done()

	env := sctx.environmentSubst()
	for _, path := range append(t.LoadEnv, sctx.loadenv...) {
		if environ, err := EnvironFromFile(sctx.variableSubst(path)); err != nil {
			return err
		} else {
			env = append(env, environ...)
		}
	}

	env = append(env, Environ(os.Expand(t.Environ, Subst(env)))...)
	for i, k := range env {
		env[i] = sctx.variableSubst(k)
	}

	if envx.Boolean(false, bw.EnvLogsVerbose) {
		log.Println("shell environment\n", strings.Join(env, "\n"))
	}

	command := sctx.variableSubst(t.Command)
	cmd := exec.CommandContext(deadline, sctx.Shell, "-c", command)
	cmd.Env = env
	cmd.Stderr = sctx.output
	cmd.Stdout = sctx.output
	cmd.Dir = stringsx.DefaultIfBlank(sctx.variableSubst(t.WorkDir), sctx.dir)

	return t.retry(sctx, func() error { return t.lenient(sctx, cmd.Run()) })
}

func (t Exec) lenient(ctx Context, err error) error {
	if (t.Lenient || ctx.lenient) && err != nil {
		fmt.Fprintln(ctx.output, "command failed, ignoring", t.Command, err)
		return nil
	}

	return err
}

func (t Exec) retry(ctx Context, do func() error) (err error) {
	retries := t.Retries
	switch retries {
	case 1:
		return do()
	case 0:
		return do()
	case -1:
		retries = math.MaxInt16
	}

	for i := int16(0); i < retries; i++ {
		if cause := do(); cause == nil {
			return nil
		} else {
			err = errorsx.Compact(err, cause)
		}

	}

	fmt.Fprintln(ctx.output, "command failed after", retries, "attempts", t.Command, err)

	return err
}

// Execute ...
func Execute(ctx context.Context, sctx Context, commands ...Exec) error {
	for _, c := range commands {
		fmt.Fprintln(sctx.output, "executing", sctx.Shell, "-c", c.Command)
		if err := c.execute(ctx, sctx); err != nil {
			return errors.Wrapf(err, "failed to execute: '%s'", c.Command)
		}
		fmt.Fprintln(sctx.output, "completed", sctx.Shell, "-c", c.Command)
	}

	return nil
}

// ParseYAML ...
func ParseYAML(r io.Reader) ([]Exec, error) {
	var (
		err     error
		raw     []byte
		results []Exec
	)

	if raw, err = io.ReadAll(r); err != nil {
		return results, errors.Wrap(err, "failed to read yaml")
	}

	if err = yaml.Unmarshal(raw, &results); err != nil {
		return results, errors.Wrap(err, "failed to decode yaml")
	}

	return results, nil
}

type logger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})
}
