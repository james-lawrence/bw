//go:build !linux
// +build !linux

package systemx

import (
	"errors"
	"log"
	"os"
	"time"
)

func FileCreatedAt(info os.FileInfo) (ctime time.Time, err error) {
	return ctime, errors.New("unable to retrieve creation time of file outside of linux")
}

func MachineID() string {
	log.Println("machine id not supported on this system")
	return ""
}
