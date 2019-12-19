package acme

import (
	"os"

	"github.com/james-lawrence/bw/internal/x/protox"
)

type solver Service

func (t solver) Present(domain, token, keyAuth string) (err error) {
	c := Challenge{Domain: domain, Token: token, Digest: keyAuth}
	return writeChallenge(Service(t).challengeFile(), c)
}

func (t solver) CleanUp(domain, token, keyAuth string) error {
	return os.Remove(Service(t).challengeFile())
}

func writeChallenge(path string, c Challenge) error {
	return protox.WriteFile(path, 0600, &c)
}

func readChallenge(path string) (c Challenge, err error) {
	if err = protox.ReadFile(path, &c); err != nil {
		return c, err
	}
	return c, err
}
