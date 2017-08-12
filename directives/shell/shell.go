package shell

import (
	"context"
	"io"
	"io/ioutil"
	"log"
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
	deadline, done := context.WithTimeout(context.Background(), (t.Timeout))
	defer done()
	command := ctx.variableSubst(t.Command)
	cmd := exec.CommandContext(deadline, ctx.Shell, "-c", command)
	cmd.Env = ctx.Environ
	cmd.Stderr = ctx.output
	cmd.Stdout = ctx.output
	return t.lenient(cmd.Run())
}

func (t Exec) lenient(err error) error {
	if t.Lenient {
		log.Println("command failed, ignoring", t.Command, err)
		return nil
	}

	return err
}

// Execute ...
func Execute(ctx Context, commands ...Exec) {
	for _, c := range commands {
		log.Println("executing", c.Command)
		if err := c.execute(ctx); err != nil {
			log.Printf("failed to execute: %s: '%s'", err, c.Command)
			return
		}
	}
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
