package acme

import (
	"crypto"
	"log"
	"path/filepath"

	"github.com/go-acme/lego/v4/registration"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/internal/x/sshx"
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
		err error
	)

	if priv, err = sshx.MaybeDecodeRSA(sshx.CachedGenerate(filepath.Join(t.Config.Root, certificatecache.DefaultACMEKey), 4096)); err != nil {
		log.Println("failed to load private key", err)
		return nil
	}

	return priv
}
