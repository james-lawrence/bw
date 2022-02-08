package interp_test

import (
	"context"
	"go/build"
	"io/ioutil"
	"log"
	"reflect"
	"strings"

	yaegi "github.com/traefik/yaegi/interp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/james-lawrence/bw/directives/interp"
)

var _ = Describe("osfix", func() {
	It("Exit should not crash the program", func() {
		c := Compiler{
			Build:   build.Default,
			Environ: []string{},
			Exports: []yaegi.Exports{},
			Log:     log.New(ioutil.Discard, "DISCARD", 0),
		}

		buf := strings.NewReader(`package main
			import "os"
			func main() {
				os.Exit(1)
			}
		`)
		err := c.Execute(context.Background(), "example.go", buf)
		Expect(err).ToNot(Succeed())
	})

	It("should override os.Getwd() the working directory", func() {
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
		Expect(c.Execute(context.Background(), "example.go", buf)).To(Succeed())
		Expect(s).To(Equal("/tmp"))
	})

	It("should disallow os.Chdir()", func() {
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
		Expect(c.Execute(context.Background(), "example.go", buf)).ToNot(Succeed())
	})
})
