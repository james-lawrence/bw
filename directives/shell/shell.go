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
	"os/exec"
	"time"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Exec ...
type Exec struct {
	Command string
	Lenient bool
	Timeout time.Duration
}

func (t Exec) execute(ctx Context) error {
	timeout := t.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	deadline, done := context.WithTimeout(context.Background(), timeout)
	defer done()

	command := ctx.variableSubst(t.Command)
	cmd := exec.CommandContext(deadline, ctx.Shell, "-c", command)
	cmd.Env = ctx.Environ
	cmd.Stderr = ctx.output
	cmd.Stdout = ctx.output

	return t.lenient(ctx, cmd.Run())
}

func (t Exec) lenient(ctx Context, err error) error {
	if t.Lenient {
		fmt.Fprintln(ctx.output, "command failed, ignoring", t.Command, err)
		return nil
	}

	return err
}

// Execute ...
func Execute(ctx Context, commands ...Exec) error {
	for _, c := range commands {
		fmt.Fprintln(ctx.output, "executing", ctx.Shell, "-c", c.Command)
		if err := c.execute(ctx); err != nil {
			return errors.Wrapf(err, "failed to execute: '%s'", c.Command)
		}
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
