package deployment

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/testingx"
)

// BenchmarkDeployLogLines the cost of the output of directive commands. the shell directive writes the output of a
// command to the deploy log one line at a time (directives/shell/log.go), so a line is a write to the log file.
func BenchmarkDeployLogLines(b *testing.B) {
	line := "npm WARN deprecated left-pad@1.3.0: use String.prototype.padStart(), a typical line of build output\n"

	b.Run("current", func(b *testing.B) {
		// the logger also writes to stderr, keep it out of the measurement.
		defer silence(b)()

		l, err := newLogger(bw.RandomID([]byte("bench")), testingx.TempDir(b), "[DEPLOY] ")
		if err != nil {
			b.Fatal(err)
		}
		defer l.Close()

		report := testingx.BenchIO(b)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Print(line)
		}
		b.StopTimer()
		report(b.N)
	})

	b.Run("proposed-without-o_sync", func(b *testing.B) {
		dst, err := os.OpenFile(filepath.Join(testingx.TempDir(b), bw.DeployLog), os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			b.Fatal(err)
		}
		defer dst.Close()

		l := log.New(dst, "[DEPLOY] ", log.Flags()^log.Lshortfile)

		report := testingx.BenchIO(b)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Print(line)
		}
		b.StopTimer()
		report(b.N)
	})

	// a whole command worth of output followed by the single sync on close, what the proposal does.
	b.Run("proposed-sync-on-close", func(b *testing.B) {
		const lines = 1000

		dir := testingx.TempDir(b)
		report := testingx.BenchIO(b)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dst, err := os.OpenFile(filepath.Join(dir, fmt.Sprintf("%d.log", i)), os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				b.Fatal(err)
			}

			l := log.New(dst, "[DEPLOY] ", log.Flags()^log.Lshortfile)
			for j := 0; j < lines; j++ {
				l.Print(line)
			}
			if err = dst.Sync(); err != nil {
				b.Fatal(err)
			}
			dst.Close()
		}
		b.StopTimer()
		report(b.N * lines)
	})
}

// silence redirects stderr to the null device, the returned function restores it.
func silence(b *testing.B) func() {
	b.Helper()

	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatal(err)
	}

	original := os.Stderr
	os.Stderr = null

	return func() {
		os.Stderr = original
		null.Close()
	}
}
