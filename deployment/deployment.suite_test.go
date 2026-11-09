package deployment_test

import (
	"io"
	"log"
	"os"
	"testing"

	"github.com/james-lawrence/bw/internal/testingx"
)

func TestMain(m *testing.M) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	log.SetOutput(io.Discard)
	code := m.Run()
	testingx.Cleanup()
	os.Exit(code)
}
