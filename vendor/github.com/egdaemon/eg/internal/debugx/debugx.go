package debugx

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"time"

	"github.com/egdaemon/eg/internal/errorsx"
	"github.com/egdaemon/eg/internal/iox"
	"github.com/egdaemon/eg/internal/md5x"
	"github.com/egdaemon/eg/internal/stringsx"
)

func genDst() (path string, dst io.WriteCloser) {
	var (
		err error
	)

	t := time.Now()
	ts := stringsx.Reverse(strconv.Itoa(int(t.Unix())))
	path = filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s-%s.trace", filepath.Base(os.Args[0]), t.Format("2006-01-02"), ts))

	if dst, err = os.Create(path); err != nil {
		log.Println(errorsx.Wrapf(err, "failed to open file: %s", path))
		log.Println("routine dump falling back to stderr")
		return "stderr", iox.WriteNopCloser(os.Stderr)
	}

	return path, dst
}

func DumpRoutinesInto(dst io.WriteCloser) error {
	return errorsx.Compact(pprof.Lookup("goroutine").WriteTo(dst, 1), dst.Close())
}

// DumpRoutines writes current goroutine stack traces to a temp file.
// and returns that files path. if for some reason a file could not be opened
// it falls back to stderr
func DumpRoutines() (path string, err error) {
	var (
		dst io.WriteCloser
	)

	path, dst = genDst()
	return path, DumpRoutinesInto(dst)
}

// DumpOnSignal runs the DumpRoutes method and prints to stderr whenever one of the provided
// signals is received.
func DumpOnSignal(ctx context.Context, sigs ...os.Signal) {
	OnSignal(func() error {
		if path, err := DumpRoutines(); err == nil {
			log.Println("dump located at:", path)
			return nil
		} else {
			return errorsx.Wrap(err, "goroutine dump failed")
		}
	})(ctx, sigs...)
}

func OnSignal(do func() error) func(context.Context, ...os.Signal) {
	return func(ctx context.Context, sigs ...os.Signal) {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, sigs...)

		for {
			select {
			case <-ctx.Done():
				return
			case s := <-signals:
				log.Println("signal processing initiated", s)
				defer log.Println("signal processing completed", s)

				if err := do(); err != nil {
					log.Println("signal processing failed", s, err)
				}
			}
		}
	}
}

func BuildID() string {
	info, _ := debug.ReadBuildInfo()
	d := md5.New()
	for _, v := range info.Settings {
		if _, err := d.Write([]byte(v.Value)); err != nil {
			panic(err)
		}
	}

	return md5x.FormatString(d)
}
