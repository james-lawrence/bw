package quorum

import (
	"bytes"
	"crypto/md5"
	"io"
	"io/ioutil"
	"log"

	"github.com/james-lawrence/bw/internal/x/errorsx"

	"github.com/pkg/errors"
)

const (
	errChecksumMismatch = errorsx.String("checksum mismatch")
)

type snapshot interface {
	Snapshot(bloom []byte) (io.ReadCloser, error)
	Checksum() ([]byte, error)
}

// NewEmptySnapshot generates an empty snapshot
func NewEmptySnapshot() EmptySnapshot {
	return EmptySnapshot{}
}

// EmptySnapshot an empty snapshot
type EmptySnapshot struct{}

// Snapshot return an empty buffer
func (t EmptySnapshot) Snapshot(bloom []byte) (io.ReadCloser, error) {
	return ioutil.NopCloser(bytes.NewReader([]byte(nil))), nil
}

// Checksum snapshot checksum
func (t EmptySnapshot) Checksum() (digest []byte, err error) {
	data := md5.Sum([]byte(nil))
	return data[:], nil
}

// NewConfig new configuration service.
func NewConfig(q *Quorum, authority snapshot, authorizations snapshot) ConfigSvc {
	return ConfigSvc{
		q:              q,
		authority:      authority,
		authorizations: authorizations,
	}
}

// ConfigSvc configuration service. agent can request configuration from quorum.
type ConfigSvc struct {
	q              *Quorum
	authority      snapshot
	authorizations snapshot
}

func (t ConfigSvc) sync(remote *Snapshot, s snapshot) (_ []byte, err error) {
	var (
		checksum []byte
		data     io.ReadCloser
	)

	// assume they don't care about it.
	if remote == nil {
		return []byte(nil), nil
	}

	if checksum, err = s.Checksum(); err != nil {
		return []byte(nil), err
	}

	if bytes.Compare(remote.Checksum, checksum) == 0 {
		return []byte(nil), nil
	}

	if data, err = s.Snapshot(remote.Bloom); err != nil {
		return []byte(nil), err
	}
	defer data.Close()

	return ioutil.ReadAll(data)
}

// Check compare the agents checksum with quorum's checksum.
func (t ConfigSvc) Check(request *Checksum, stream Configuration_CheckServer) (err error) {
	var (
		data []byte
	)

	if _, err = t.q.quorumOnly(); err != nil {
		return errors.Wrap(err, "check")
	}

	if data, err = t.sync(&Snapshot{Checksum: request.Authority}, t.authority); err != nil {
		log.Println("failed to sync authority")
	}

	if err = stream.Send(&Config{Type: &Config_Authority{Authority: data}}); err != nil {
		return err
	}

	if data, err = t.sync(request.Authorizations, t.authorizations); err != nil {
		log.Println("failed to sync authority")
	}

	if err = stream.Send(&Config{Type: &Config_Authorizations{Authorizations: data}}); err != nil {
		return err
	}

	if err = stream.Send(&Config{Type: &Config_Checksum{Checksum: request}}); err != nil {
		return err
	}

	return nil
}
