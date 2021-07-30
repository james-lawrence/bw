package acme

import (
	"crypto"
	"log"
	"path/filepath"

	"github.com/go-acme/lego/v4/registration"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/internal/rsax"
)

const (
	regfilename = "acme.registration.json"
)

type account struct {
	agent.Config
	certificatecache.ACMEConfig
}

func (t account) GetEmail() string {
	return t.ACMEConfig.Email
}

func (t account) GetRegistration() (reg *registration.Resource) {
	return readRegistration(t.Config)
}

func (t account) GetPrivateKey() (priv crypto.PrivateKey) {
	var (
		err    error
		secret = []byte(t.ACMEConfig.Secret)
	)

	if len(secret) > 0 {
		if priv, err = rsax.MaybeDecode(rsax.CachedAutoDeterministic(secret, filepath.Join(t.Config.Root, certificatecache.DefaultACMEKey))); err != nil {
			log.Println("failed to load private key", err)
			return nil
		}
	} else {
		log.Println("WARNING: acme config is missing a secret, add `secret: \"examplesecret\"` to the configuration")
		if priv, err = rsax.MaybeDecode(rsax.CachedAuto(filepath.Join(t.Config.Root, certificatecache.DefaultACMEKey))); err != nil {
			log.Println("failed to load private key", err)
			return nil
		}
	}

	return priv
}
