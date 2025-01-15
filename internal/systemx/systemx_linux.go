package systemx

import (
	"errors"
	"log"
	"os"
	"strings"
	"syscall"
	"time"
)

func MachineID() string {
	var (
		err error
		raw []byte
	)

	if raw, err = os.ReadFile("/etc/machine-id"); err != nil {
		log.Println("failed to read machine id, defaulting to empty string", err)
		return ""
	}

	return strings.TrimSpace(string(raw))
}

// FileCreatedAt determine the creation time of a file.
func FileCreatedAt(info os.FileInfo) (ctime time.Time, err error) {
	var (
		ok   bool
		stat *syscall.Stat_t
	)

	sys := info.Sys()

	if sys == nil {
		return ctime, errors.New("missing system information, unable to retrieve ctime")
	}

	if stat, ok = sys.(*syscall.Stat_t); !ok {
		return ctime, errors.New("missing system information, unable to retrieve ctime")
	}

	return time.Unix(int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec)), nil
}
