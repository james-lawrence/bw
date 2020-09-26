package interp_test

import (
	"context"
	"go/build"
	"reflect"
	"strings"

	yaegi "github.com/traefik/yaegi/interp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/james-lawrence/bw/directives/interp"
)

var _ = Describe("Interp", func() {
	It("should execute a function", func() {
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
		Expect(c.Execute(context.Background(), "example.go", buf)).To(Succeed())
		Expect(i).To(Equal(1))
	})

	It("should be able to read the environment", func() {
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
		Expect(c.Execute(context.Background(), "example.go", buf)).To(Succeed())
		Expect(s).To(Equal("foo"))
	})

	It("should handle os.Exit gracefully", func() {
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
		Expect(err).NotTo(Succeed())
	})

	It("should gracefully handle panics", func() {
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
		Expect(err).ToNot(Succeed())
	})
})
