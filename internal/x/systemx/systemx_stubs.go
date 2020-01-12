// +build !linux

package systemx

import (
	"errors"
	"time"
)

func FileCreatedAt(info os.Info) (ctime time.Time, err error) {
	return ctime, errors.New("unable to retrieve creation time of file outside of linux")
}
