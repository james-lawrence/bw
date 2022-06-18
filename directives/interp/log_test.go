package interp_test

import (
	"bytes"
	"context"
	"go/build"
	"io"
	"log"
	"strings"

	yaegi "github.com/traefik/yaegi/interp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/james-lawrence/bw/directives/interp"
)

var _ = Describe("log", func() {
	Describe("SetFlags", func() {
		It("should error out", func() {
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
			Expect(err).ToNot(Succeed())
		})
	})

	Describe("Println", func() {
		It("should log the message", func() {
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
			Expect(err).To(Succeed())
			Expect(logs.String()).To(Equal("Hello World\n"))
		})
	})

	Describe("Fatalln", func() {
		It("should log the message", func() {
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
			Expect(err).ToNot(Succeed())
			Expect(logs.String()).To(Equal("Hello World\n"))
		})
	})
})
