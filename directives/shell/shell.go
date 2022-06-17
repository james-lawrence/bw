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
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"github.com/james-lawrence/bw/internal/x/timex"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Exec ...
type Exec struct {
	Command string
	Lenient bool
	Timeout time.Duration
	Environ string
	LoadEnv []string `yaml:"loadenv"`
}

func (t Exec) execute(ctx context.Context, sctx Context) error {
	timeout := timex.DurationOrDefault(t.Timeout, sctx.timeout)
	deadline, done := context.WithTimeout(ctx, timeout)
	defer done()

	env := sctx.environmentSubst()
	for _, path := range t.LoadEnv {
		if environ, err := EnvironFromFile(path); err != nil {
			return err
		} else {
			env = append(env, environ...)
		}
	}

	env = append(env, Environ(os.Expand(t.Environ, Subst(env)))...)

	command := sctx.variableSubst(t.Command)
	cmd := exec.CommandContext(deadline, sctx.Shell, "-c", command)
	cmd.Env = env
	cmd.Stderr = sctx.output
	cmd.Stdout = sctx.output
	cmd.Dir = sctx.dir

	return t.lenient(sctx, cmd.Run())
}

func (t Exec) lenient(ctx Context, err error) error {
	if (t.Lenient || ctx.lenient) && err != nil {
		fmt.Fprintln(ctx.output, "command failed, ignoring", t.Command, err)
		return nil
	}

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

	if raw, err = ioutil.ReadAll(r); err != nil {
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
