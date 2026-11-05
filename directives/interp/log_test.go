package interp_test

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/require"
	"go/build"
	"io"
	"log"
	"strings"
	"testing"

	yaegi "github.com/traefik/yaegi/interp"

	. "github.com/james-lawrence/bw/directives/interp"
)

func TestLog(t *testing.T) {
	t.Run("SetFlags", func(t *testing.T) {
		t.Run("should error out", func(t *testing.T) {
			c := Compiler{
				Build:   build.Default,
				Environ: []string{},
				Exports: []yaegi.Exports{},
				Log:     log.New(io.Discard, "DISCARD", 0),
			}

			buf := strings.NewReader(`package main
				import "log"
				func main() {
					log.SetFlags(1)
				}
			`)
			err := c.Execute(context.Background(), "example.go", buf)
			require.Error(t, err)
		})
	})

	t.Run("Println", func(t *testing.T) {
		t.Run("should log the message", func(t *testing.T) {
			logs := bytes.NewBufferString("")
			c := Compiler{
				Build:   build.Default,
				Environ: []string{},
				Exports: []yaegi.Exports{},
				Log:     log.New(logs, "", 0),
			}

			buf := strings.NewReader(`package main
				import "log"
				func main() {
					log.Println("Hello World")
				}
			`)

			err := c.Execute(context.Background(), "example.go", buf)
			require.NoError(t, err)
			require.Equal(t, "Hello World\n", logs.String())
		})
	})

	t.Run("Fatalln", func(t *testing.T) {
		t.Run("should log the message", func(t *testing.T) {
			logs := bytes.NewBufferString("")
			c := Compiler{
				Build:   build.Default,
				Environ: []string{},
				Exports: []yaegi.Exports{},
				Log:     log.New(logs, "", 0),
			}

			buf := strings.NewReader(`package main
				import "log"
				func main() {
					log.Fatalln("Hello World")
				}
			`)

			err := c.Execute(context.Background(), "example.go", buf)
			require.Error(t, err)
			require.Equal(t, "Hello World\n", logs.String())
		})
	})
}
