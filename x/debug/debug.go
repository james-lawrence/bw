package debug

import (
	"fmt"
	"log"
	"os"
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
