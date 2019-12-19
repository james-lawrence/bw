package sshx

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"

	"golang.org/x/crypto/ssh"

	"github.com/james-lawrence/bw/internal/x/systemx"
	"github.com/pkg/errors"
)

const (
	defaultBits = 8096 // 8096 bit keysize, reasonable default.
)

// Auto generates a ssh key using package defined defaults.
func Auto() (pkey []byte, err error) {
	return Generate(defaultBits)
}

// CachedAuto loads/generates an SSH key at the provided filepath.
func CachedAuto(path string) (pkey []byte, err error) {
	if systemx.FileExists(path) {
		return ioutil.ReadFile(path)
	}

	if pkey, err = Auto(); err != nil {
		return nil, err
	}

	if err = ioutil.WriteFile(path, pkey, 0600); err != nil {
		return nil, err
	}

	return pkey, nil
}

// UnsafeAuto generates a ssh key using unsafe defaults, this method is used to
// generate an ssh key quickly for cases were we do not care about safety, i.e.
// tests.
func UnsafeAuto() (pkey []byte, err error) {
	return Generate(128)
}

// Generate a RSA private key with the given bits size, returns the pem encoded bytes.
func Generate(bits int) (encoded []byte, err error) {
	var (
		pkey *rsa.PrivateKey
	)

	if pkey, err = private(bits); err != nil {
		return encoded, err
	}

	// Get ASN.1 DER format
	marshalled := x509.MarshalPKCS1PrivateKey(pkey)

	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: marshalled,
	}), nil
}

// generatePrivateKey creates a RSA Private Key of specified byte size
func private(bits int) (k *rsa.PrivateKey, err error) {
	// Private Key generation
	if k, err = rsa.GenerateKey(rand.Reader, bits); err != nil {
		return k, err
	}

	// Validate Private Key
	return k, k.Validate()
}

// PublicKey returns a public key from the pem encoded private key.
func PublicKey(pemkey []byte) (pub []byte, err error) {
	var (
		pkey   *rsa.PrivateKey
		pubkey ssh.PublicKey
	)

	blk, _ := pem.Decode(pemkey) // assumes a single valid pem encoded key.

	if pkey, err = x509.ParsePKCS1PrivateKey(blk.Bytes); err != nil {
		return pub, err
	}

	if pubkey, err = ssh.NewPublicKey(&pkey.PublicKey); err != nil {
		return pub, err
	}

	return ssh.MarshalAuthorizedKey(pubkey), nil
}

// IsNoKeyFound check if ssh key is not found.
func IsNoKeyFound(err error) bool {
	return err.Error() == "ssh: no key found"
}

// DecodeRSA decode a RSA private key.
func DecodeRSA(encoded []byte) (priv *rsa.PrivateKey, err error) {
	b, _ := pem.Decode(encoded)
	if priv, err = x509.ParsePKCS1PrivateKey(b.Bytes); err != nil {
		return nil, errors.WithStack(err)
	}

	return priv, nil
}

// MaybeDecodeRSA decodes RSA from an encoded array and possible error.
func MaybeDecodeRSA(encoded []byte, err error) (priv *rsa.PrivateKey, _ error) {
	if err != nil {
		return priv, err
	}
	return DecodeRSA(encoded)
}
