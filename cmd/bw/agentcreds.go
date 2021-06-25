package main

import (
	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/james-lawrence/bw/internal/sshx"
	"github.com/james-lawrence/bw/notary"
)

func genkey(k []byte) (dpriv []byte, dpub []byte, err error) {
	if dpriv, err = rsax.AutoDeterministic(k); err != nil {
		return dpriv, dpub, err
	}

	if dpub, err = sshx.PublicKey(dpriv); err != nil {
		return dpriv, dpub, err
	}

	return dpriv, dpub, err
}

func generatecredentials(config agent.Config, n notary.Composite) (ss notary.Signer, err error) {
	var (
		ring  *memberlist.Keyring
		dpriv []byte
		dpub  []byte
	)

	if ring, err = config.Keyring(); err != nil {
		return ss, err
	}

	for _, k := range ring.GetKeys() {
		if _, dpub, err = genkey(k); err != nil {
			return ss, err
		}

		if _, err = n.Insert(notary.AgentGrant(dpub)); err != nil {
			return ss, err
		}
	}

	if dpriv, _, err = genkey(ring.GetPrimaryKey()); err != nil {
		return ss, err
	}

	if ss, err = notary.NewSigner(dpriv); err != nil {
		return ss, err
	}

	return ss, err
}
