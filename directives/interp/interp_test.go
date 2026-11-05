package interp_test

import (
	"context"
	"github.com/stretchr/testify/require"
	"go/build"
	"io"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"

	yaegi "github.com/traefik/yaegi/interp"

	. "github.com/james-lawrence/bw/directives/interp"
)

func TestInterp(t *testing.T) {
	t.Run("should execute a function", func(t *testing.T) {
		i := 0
		inc := func() {
			i++
		}

		c := Compiler{
			Build:   build.Default,
			Environ: []string{},
			Exports: []yaegi.Exports{
				yaegi.Exports{
					"example": map[string]reflect.Value{
						"Increment": reflect.ValueOf((func())(inc)),
					},
				},
			},
		}
		buf := strings.NewReader(`package main
			import (
				"example"
				"bw/interp/env"
				"log"
			)
			func main() {
				example.Increment()
			}
		`)
		require.NoError(t, c.Execute(context.Background(), "example.go", buf))
		require.Equal(t, 1, i)
	})

	t.Run("should be able to read the environment", func(t *testing.T) {
		s := ""
		inc := func(in string) {
			s = in
		}

		c := Compiler{
			Build:   build.Default,
			Environ: []string{"BEARDED_WOOKIE=foo", "Bar=bar"},
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
				"bw/interp/env"
				"log"
			)

			func main() {
				example.String(env.BEARDED_WOOKIE)
			}
		`)
		require.NoError(t, c.Execute(context.Background(), "example.go", buf))
		require.Equal(t, "foo", s)
	})

	t.Run("should handle os.Exit gracefully", func(t *testing.T) {
		c := Compiler{
			Build:   build.Default,
			Environ: []string{},
			Exports: []yaegi.Exports{},
		}

		buf := strings.NewReader(`package main
			import (
				"os"
			)

			func main() {
				os.Exit(2)
			}
		`)
		err := c.Execute(context.Background(), "example.go", buf)
		require.Error(t, err)
	})

	t.Run("should gracefully handle panics", func(t *testing.T) {
		c := Compiler{
			Build:   build.Default,
			Environ: []string{},
			Exports: []yaegi.Exports{},
		}

		buf := strings.NewReader(`package main
			func main() {
				panic("fail")
			}
		`)
		err := c.Execute(context.Background(), "example.go", buf)
		require.Error(t, err)
	})
}

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}
