package debugx

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"time"

	"bitbucket.org/jatone/bearded-wookie/x/stringsx"
)

// DumpRoutines writes current goroutine stack traces to a temp file.
// and returns that files path.
func DumpRoutines() (string, error) {
	t := time.Now()
	ts := stringsx.Reverse(strconv.Itoa(int(t.Unix())))
	path := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s-mailman.trace", t.Format("2006-01-02"), ts))

	out, err := os.Create(path)
	if err != nil {
		log.Printf("failed to open file (%s):%s\n", path, err)
		return "", err
	}
	if err = pprof.Lookup("goroutine").WriteTo(out, 1); err != nil {
		return "", err
	}

	return path, nil
}

// DumpOnSignal runs the DumpRoutes method and prints to stderr whenever one of the provided
// signals is received.
func DumpOnSignal(ctx context.Context, sigs ...os.Signal) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, sigs...)

	for {
		select {
		case <-ctx.Done():
			return
		case _ = <-signals:
			if path, err := DumpRoutines(); err == nil {
				log.Println("dump located at:", path)
			} else {
				log.Println("failed to dump routines:", err)
			}
		}
	}
}
