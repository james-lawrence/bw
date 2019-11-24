package notary

import (
	"crypto/md5"
	"crypto/rand"
	"strconv"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"

	"github.com/james-lawrence/bw/internal/x/errorsx"
)

// this method is sacred, it must remain backwards compatible between versions.
// this is why we didn't use proto.Marshal to generate the binary data for the signature.
func genSignatureData(t *Token) (b []byte, err error) {
	failed := func(n int, err error) error {
		return err
	}

	digest := md5.New()

	err = errorsx.Compact(
		failed(digest.Write(t.ID)),
		failed(digest.Write([]byte(t.Fingerprint))),
		failed(digest.Write([]byte(strconv.Itoa(int(t.Issued))))),
		failed(digest.Write([]byte(strconv.Itoa(int(t.Expires))))),
	)

	return b, errors.Wrap(err, "failed to generate signature")
}

// GeneratureSignature generate a signature for the provided token.
func genSignature(k ssh.Signer, t Token) (s Signature, err error) {
	var (
		b  []byte
		ss *ssh.Signature
	)

	if b, err = genSignatureData(&t); err != nil {
		return s, errors.Wrap(err, "failed to generate signature")
	}

	if ss, err = k.Sign(rand.Reader, b); err != nil {
		return s, errors.Wrap(err, "failed to generate signature")
	}

	return Signature{
		Format: ss.Format,
		Data:   ss.Blob,
	}, nil
}

func (t Signature) sig() *ssh.Signature {
	return &ssh.Signature{
		Format: t.Format,
		Blob:   t.Data,
	}
}
