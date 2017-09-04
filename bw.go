package bw

import (
	"crypto/md5"
	"encoding/base32"
	"io"
	"math/rand"
	"time"

	"github.com/pkg/errors"
)

//go:generate protoc -I=.protocol --go_out=plugins=grpc:deployment/agent .protocol/agent.proto

// RandomID a random identifier.
type RandomID []byte

func (t RandomID) String() string {
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(t)
}

// GenerateID generates a random ID.
func GenerateID() (_ignored RandomID, err error) {
	const n = 1024
	randIDHash := md5.New()
	if _, err = io.CopyN(randIDHash, rand.New(rand.NewSource(time.Now().Unix())), n); err != nil {
		return _ignored, errors.Wrap(err, "failed generating data for deploymentID")
	}

	return randIDHash.Sum(nil), nil
}
