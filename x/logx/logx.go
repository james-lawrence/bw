package logx

import "log"

// MaybeLog ...
func MaybeLog(err error) error {
	if err != nil {
		log.Println(err)
	}

	return err
}
