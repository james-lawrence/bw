package sshx

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

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
