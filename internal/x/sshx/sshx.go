package sshx

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

const (
	defaultBits = 8096 // 8096 bit keysize, reasonable default.
)

// Auto generates a ssh key using package defined defaults.
func Auto() (pkey []byte, err error) {
	return Generate(defaultBits)
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
