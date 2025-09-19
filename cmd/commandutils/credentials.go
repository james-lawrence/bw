package commandutils

import (
	"log"
	"path/filepath"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/md5x"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/james-lawrence/bw/internal/sshx"
	"github.com/james-lawrence/bw/notary"
)

func genkey(config agent.Config, k []byte) (dpriv []byte, dpub []byte, err error) {
	cachepath := filepath.Join(config.Root, bw.DirCache, "tokens", md5x.Digest(k))
	if dpriv, err = rsax.CachedAutoDeterministic(k, cachepath); err != nil {
		return dpriv, dpub, err
	}

	if dpub, err = sshx.PublicKey(dpriv); err != nil {
		return dpriv, dpub, err
	}

	return dpriv, dpub, err
}

func Generatecredentials(config agent.Config, n notary.Composite) (ss notary.Signer, err error) {
	var (
		ring  *memberlist.Keyring
		dpriv []byte
		dpub  []byte
	)

	log.Println("generating credentials initiated")
	defer log.Println("generating credentials completed")

	if config.Credentials.PresharedKey != "" {
		agentID := config.Name
		if agentID == "" {
			agentID = "unknown"
		}

		log.Printf("using preshared key for agent credentials: %s", agentID)
		return notary.NewAgentPresharedKeySigner(config.Root, config.Credentials.PresharedKey, agentID)
	}

	if ring, err = config.Keyring(); err != nil {
		return ss, err
	}

	for _, k := range ring.GetKeys() {
		if _, dpub, err = genkey(config, k); err != nil {
			return ss, err
		}

		if _, err = n.Insert(notary.AgentGrant(dpub)); err != nil {
			return ss, err
		}
	}

	if dpriv, _, err = genkey(config, ring.GetPrimaryKey()); err != nil {
		return ss, err
	}

	if ss, err = notary.NewSigner(dpriv); err != nil {
		return ss, err
	}

	return ss, err
}
