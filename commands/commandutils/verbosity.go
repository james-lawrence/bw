package commandutils

import (
	"io/ioutil"
	"log"
	"os"
)

const (
	// VerbosityQuiet default verbosity setting, minimal output.
	VerbosityQuiet = "quiet"
	// VerbosityStack print stack trace when an error occurs.
	VerbosityStack = "stack"
)

// ConfigLog configures logs based on the verbosity
func ConfigLog(v string) {
	switch v {
	case VerbosityStack:
		Verbose = log.New(os.Stderr, "[Verbose] ", log.Flags()|log.Lshortfile)
		fallthrough
	default:
		log.SetFlags(log.Flags() | log.Lshortfile)
	}
}

// Fatalln returns a string format based on the verbosity.
func Fatalln(v string, err error) {
	switch v {
	case VerbosityStack:
		Verbose.Fatalf("%+v\n", err)
	default:
		log.Fatalln(err)
	}
}

var (
	// Verbose logger
	Verbose = log.New(ioutil.Discard, "[Verbose] ", log.Flags()|log.Lshortfile)
)
