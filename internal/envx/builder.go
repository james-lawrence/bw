package envx

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/james-lawrence/bw/internal/debugx"
	"github.com/james-lawrence/bw/internal/errorsx"
)

// format an environment variable in k=v, surrounding double quotes are stripped from the value.
// it doesn't currently escape values.
func format(k string, v string) string {
	return fmt.Sprintf("%s=%s", k, strings.TrimPrefix(strings.TrimSuffix(v, "\""), "\""))
}

func fromReader(r io.Reader) (environ []string, err error) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		t := scanner.Text()
		k, v, ok := strings.Cut(t, "=")
		if !ok {
			debugx.Println("skipping malformed env var", t)
			continue
		}

		environ = append(environ, format(k, v))
	}

	return environ, scanner.Err()
}

func fromPath(n string) (environ []string, err error) {
	env, err := os.Open(n)
	if os.IsNotExist(err) {
		return environ, nil
	}
	if err != nil {
		return nil, err
	}
	defer env.Close()

	return fromReader(env)
}

// Build an environment, in the form of key=value strings, from various sources.
// errors are accumulated and reported by the methods producing output.
func Build() *Builder {
	return &Builder{}
}

type Builder struct {
	environ []string
	failed  error
}

// CopyTo writes the environment to w, one variable per line.
func (t Builder) CopyTo(w io.Writer) error {
	if t.failed != nil {
		return t.failed
	}

	for _, e := range t.environ {
		if _, err := fmt.Fprintf(w, "%s\n", e); err != nil {
			return errorsx.Wrapf(err, "unable to write environment variable: %s", e)
		}
	}

	return nil
}

// WriteTo writes the environment to the file at path.
func (t Builder) WriteTo(path string) error {
	var (
		buf bytes.Buffer
	)

	if err := t.CopyTo(&buf); err != nil {
		return err
	}

	return os.WriteFile(path, buf.Bytes(), 0600)
}

// Environ returns the environment and any error encountered building it.
func (t Builder) Environ() ([]string, error) {
	return t.environ, t.failed
}

// Var sets a single variable to a value.
func (t *Builder) Var(k, v string) *Builder {
	t.environ = append(t.environ, fmt.Sprintf("%s=%s", k, v))
	return t
}

// FromPath extracts environment variables from files on disk.
// missing files are treated as noops.
func (t *Builder) FromPath(p ...string) *Builder {
	for _, n := range p {
		tmp, err := fromPath(n)
		t.environ = append(t.environ, tmp...)
		t.failed = errors.Join(t.failed, err)
	}
	return t
}

// FromReader extracts environment variables from an io.Reader.
// the format is the standard .env file format.
func (t *Builder) FromReader(r io.Reader) *Builder {
	tmp, err := fromReader(r)
	t.environ = append(t.environ, tmp...)
	t.failed = errors.Join(t.failed, err)
	return t
}

// FromEnviron adds key=value strings verbatim.
func (t *Builder) FromEnviron(environ ...string) *Builder {
	t.environ = append(t.environ, environ...)
	return t
}

// FromEnv extracts the named keys from the process environment.
// keys that are not set are omitted. keys set to an empty value are kept as k=.
func (t *Builder) FromEnv(keys ...string) *Builder {
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			t.environ = append(t.environ, format(k, v))
		}
	}

	return t
}

// Only keeps just the specified keys. when a key is repeated the last value is kept.
func (t *Builder) Only(keys ...string) *Builder {
	uniq := make(map[string]string, len(t.environ))
	for _, v := range t.environ {
		if key, _, ok := strings.Cut(v, "="); ok {
			uniq[key] = v
		}
	}

	vars := make([]string, 0, len(keys))
	for _, k := range keys {
		if v, ok := uniq[k]; ok {
			vars = append(vars, v)
		}
	}

	t.environ = vars
	return t
}

// Drop removes every occurrence of the specified keys.
func (t *Builder) Drop(keys ...string) *Builder {
	dropped := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		dropped[k] = struct{}{}
	}

	vars := make([]string, 0, len(t.environ))
	for _, s := range t.environ {
		k, _, ok := strings.Cut(s, "=")
		if !ok {
			debugx.Println("malformed environment variable", s)
		}

		if _, ok := dropped[k]; ok {
			debugx.Println("dropping", k)
			continue
		}

		vars = append(vars, s)
	}

	t.environ = vars
	return t
}

// Setenv replaces the value of every occurrence of k. variables that are not already
// present are not added.
func (t *Builder) Setenv(k, v string) error {
	for idx, s := range t.environ {
		if key, _, ok := strings.Cut(s, "="); ok && key == k {
			t.environ[idx] = fmt.Sprintf("%s=%s", k, v)
		}
	}

	return nil
}

// Append v to the value of k using sep as the separator. k is added when it is not present.
func (t *Builder) Append(k, v, sep string) error {
	found := false
	for idx, s := range t.environ {
		if key, value, ok := strings.Cut(s, "="); ok && key == k {
			found = true
			t.environ[idx] = fmt.Sprintf("%s=%s%s%s", k, value, sep, v)
		}
	}

	if !found {
		t.environ = append(t.environ, fmt.Sprintf("%s=%s", k, v))
	}

	return nil
}

// Unsetenv removes every occurrence of k.
func (t *Builder) Unsetenv(k string) error {
	vars := make([]string, 0, len(t.environ))
	for _, s := range t.environ {
		if key, _, ok := strings.Cut(s, "="); ok && key == k {
			continue
		}

		vars = append(vars, s)
	}

	t.environ = vars
	return nil
}
