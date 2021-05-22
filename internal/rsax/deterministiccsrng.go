package rsax

import (
	"crypto/sha512"
	"hash"
	"io"
)

// NewSHA512CSPRNG generate a csprng using sha512.
func NewSHA512CSPRNG(seed []byte) io.Reader {
	return &sha512csprng{
		seed: seed,
		hash: sha512.New(),
	}
}

type sha512csprng struct {
	hash  hash.Hash
	seed  []byte
	state []byte
}

func (t *sha512csprng) Read(b []byte) (n int, err error) {
	if t.state == nil {
		if t.state, err = t.update(t.seed); err != nil {
			return n, err
		}
	}

	for i := len(b); i > 0; i = i - len(t.state) {
		random := t.state
		if i < len(t.state) {
			random = t.state[:i]
		}

		n += copy(b[n:], random)

		if t.state, err = t.update(t.state); err != nil {
			return n, err
		}
	}

	return n, nil
}

func (t *sha512csprng) update(state []byte) ([]byte, error) {
	_, err := t.hash.Write(state)
	return t.hash.Sum(nil), err
}
