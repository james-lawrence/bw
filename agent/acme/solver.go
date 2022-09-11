package acme

import (
	"os"

	"github.com/james-lawrence/bw/internal/protox"
)

type solver DiskCache

func (t solver) Present(domain, token, keyAuth string) (err error) {
	c := &Challenge{Domain: domain, Token: token, Digest: keyAuth}
	return writeChallenge(DiskCache(t).challengeFile(), c)
}

func (t solver) CleanUp(domain, token, keyAuth string) error {
	return os.Remove(DiskCache(t).challengeFile())
}

func writeChallenge(path string, c *Challenge) error {
	return protox.WriteFile(path, 0600, c)
}

func readChallenge(path string) (c *Challenge, err error) {
	c = &Challenge{}
	if err = protox.ReadFile(path, c); err != nil {
		return c, err
	}
	return c, err
}
