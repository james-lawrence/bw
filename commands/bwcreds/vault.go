package main

import (
	"io/ioutil"
	"log"
	"os/user"
	"path/filepath"

	"github.com/alecthomas/kingpin"
	"github.com/hashicorp/vault/api"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/commands/commandutils"
	"github.com/james-lawrence/bw/x/stringsx"
	"github.com/pkg/errors"
)

type vaultCreds struct {
	environment      string
	path             string
	commonName       string
	agentCredentials bool
}

func (t *vaultCreds) configure(parent *kingpin.CmdClause) {
	commandutils.EnvironmentArg(parent).StringVar(&t.environment)
	parent.Arg("path", "path to vault private key interface").StringVar(&t.path)
	parent.Arg("common-name", "common name for certificate, usually a domain name").StringVar(&t.commonName)
	parent.Flag("agent", "generate credentials for an agent, mainly used on servers").Default("false").BoolVar(&t.agentCredentials)
	parent.Action(t.generate)
}

func (t *vaultCreds) generate(ctx *kingpin.ParseContext) (err error) {
	var (
		client      *api.Client
		credentials *api.Secret
		config      *api.Config
	)

	if config = api.DefaultConfig(); config.Error != nil {
		return errors.WithStack(config.Error)
	}

	if client, err = api.NewClient(config); err != nil {
		return errors.WithStack(err)
	}
	client.SetToken(stringsx.DefaultIfBlank(client.Token(), t.readTokenFile()))

	payload := map[string]interface{}{
		"common_name": t.commonName,
	}

	if credentials, err = client.Logical().Write(t.path, payload); err != nil {
		return errors.WithStack(err)
	}

	log.Println("credentials fingerprint", credentials.Data["serial_number"])

	capath := filepath.Join(bw.DefaultLocation(t.environment, ""), agent.DefaultTLSCertCA)
	keypath := filepath.Join(bw.DefaultLocation(t.environment, ""), agent.DefaultTLSKeyClient)
	certpath := filepath.Join(bw.DefaultLocation(t.environment, ""), agent.DefaultTLSCertClient)
	if t.agentCredentials {
		keypath = filepath.Join(bw.DefaultLocation(t.environment, ""), agent.DefaultTLSKeyServer)
		certpath = filepath.Join(bw.DefaultLocation(t.environment, ""), agent.DefaultTLSCertServer)
	}

	log.Println("writing private key", keypath)
	if err = ioutil.WriteFile(keypath, []byte(credentials.Data["private_key"].(string)), 0600); err != nil {
		return errors.Wrapf(err, "failed to write private key to %s", keypath)
	}

	log.Println("writing certificate", certpath)
	if err = ioutil.WriteFile(certpath, []byte(credentials.Data["certificate"].(string)), 0600); err != nil {
		return errors.Wrapf(err, "failed to write certificate to %s", certpath)
	}

	log.Println("writing authority certificate", capath)
	if err = ioutil.WriteFile(capath, []byte(credentials.Data["issuing_ca"].(string)), 0600); err != nil {
		return errors.Wrapf(err, "failed to write certificate authority to %s", capath)
	}

	return nil
}

func (t vaultCreds) readTokenFile() string {
	var (
		err error
		u   *user.User
		raw []byte
	)

	if u, err = user.Current(); err != nil {
		commandutils.Verbose.Println("failed to lookup user, vault token not loaded from file", err)
		return ""
	}

	if raw, err = ioutil.ReadFile(filepath.Join(u.HomeDir, ".vault-token")); err != nil {
		commandutils.Verbose.Println("failed to read vault token from file", err)
		return ""
	}

	return string(raw)
}
