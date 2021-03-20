package muxer

import "crypto/md5"

// Protocol uuid.
type Protocol [16]byte

func Proto(name string) Protocol {
	return md5.Sum([]byte(name))
}
