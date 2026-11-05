package interp_test

import (
	"context"
	"github.com/stretchr/testify/require"
	"go/build"
	"io"
	"log"
	"reflect"
	"strings"
	"testing"

	yaegi "github.com/traefik/yaegi/interp"

	. "github.com/james-lawrence/bw/directives/interp"
)

func TestOsfix(t *testing.T) {
	t.Run("Exit should not crash the program", func(t *testing.T) {
		c := Compiler{
			Build:   build.Default,
			Environ: []string{},
			Exports: []yaegi.Exports{},
			Log:     log.New(io.Discard, "DISCARD", 0),
		}

		buf := strings.NewReader(`package main
			import "os"
			func main() {
				os.Exit(1)
			}
		`)
		err := c.Execute(context.Background(), "example.go", buf)
		require.Error(t, err)
	})

	t.Run("should override os.Getwd() the working directory", func(t *testing.T) {
		s := ""
		inc := func(in string) {
			s = in
		}

		c := Compiler{
			Build:            build.Default,
			WorkingDirectory: "/tmp",
			Environ:          []string{"BEARDED_WOOKIE=foo", "Bar=bar"},
			Exports: []yaegi.Exports{
				yaegi.Exports{
					"example": map[string]reflect.Value{
						"String": reflect.ValueOf((func(string))(inc)),
					},
				},
			},
		}

		buf := strings.NewReader(`package main
			import (
				"example"
				"os"
			)

			func main() {
				dir, _ := os.Getwd()
				example.String(dir)
			}
		`)
		require.NoError(t, c.Execute(context.Background(), "example.go", buf))
		require.Equal(t, "/tmp", s)
	})

	t.Run("should disallow os.Chdir()", func(t *testing.T) {
		c := Compiler{
			Build:            build.Default,
			WorkingDirectory: "/tmp",
			Environ:          []string{"BEARDED_WOOKIE=foo", "Bar=bar"},
		}

		buf := strings.NewReader(`package main
			import (
				"example"
				"os"
			)

			func main() {
				if err := os.Chdir("foo"); err != nil {
					log.Fatalln(err)
				}
			}
		`)
		require.Error(t, c.Execute(context.Background(), "example.go", buf))
	})
}
