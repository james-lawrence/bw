package notary

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/x/sshx"
	"github.com/james-lawrence/bw/internal/x/systemx"
)

// NewFromFile load notary configuration from a file.
func NewFromFile(root, path string) (c Composite, err error) {
	var (
		in io.ReadCloser
	)

	if in, err = os.Open(path); err != nil {
		return c, errors.Wrapf(err, "failed to read configuration from: %s", path)
	}
	defer in.Close()

	if c, err = NewFrom(root, in); err != nil {
		return c, errors.Wrapf(err, "failed to read configuration from: %s", path)
	}

	return c, nil
}

// NewFrom parse the configuration from an io.Reader.
func NewFrom(root string, in io.Reader) (c Composite, err error) {
	var (
		nc        = notary{}
		directory = NewDirectory(filepath.Join(root, "dynamic"))
		bin       []byte
	)

	if bin, err = ioutil.ReadAll(in); err != nil {
		return c, err
	}

	if err = bw.ExpandAndDecode(bin, &nc); err != nil {
		return c, err
	}

	buckets := make([]storage, 0, len(nc.Config.Authority))
	log.Println("loading authorizations", nc.Config.Authority)
	for _, p := range nc.Config.Authority {
		var (
			b storage
		)

		if b, err = newFile(p); err != nil {
			return c, errors.Wrapf(err, "failed to initialize: %s", p)
		}

		buckets = append(buckets, b)
	}

	return NewComposite(directory, buckets...), nil
}

type notary struct {
	Config nconfig `yaml:"notary"`
}

type nconfig struct {
	Authority []string `yaml:"authority"`
}

func genKey(root, fingerprint string) string {
	return filepath.Join(root, bw.DirAuthorizations, fingerprint)
}

func genFingerprint(d []byte) string {
	digest := sha256.Sum256(d)
	return hex.EncodeToString(digest[:])
}

func defaultAuthorizationsPath() []string {
	var (
		err error
		u   *user.User
	)

	if u, err = user.Current(); err != nil {
		log.Println("failed to load current user for authorized keys", err)
		return []string{}
	}

	return []string{filepath.Join(u.HomeDir, ".ssh", "authorized_keys")}
}

func loadAuthorizedKeys(s storage, path string) (err error) {
	var (
		encoded []byte
	)

	if !systemx.FileExists(path) {
		log.Println("not loading", path, "does not exist")
		return nil
	}

	log.Println("loading authorization keys from", path)

	if encoded, err = ioutil.ReadFile(path); err != nil {
		return err
	}

	for len(encoded) != 0 {
		var (
			key ssh.PublicKey
		)

		if key, _, _, encoded, err = ssh.ParseAuthorizedKey(encoded); err != nil {
			if sshx.IsNoKeyFound(err) {
				continue
			}
			log.Println(err)
			continue
		}

		g := Grant{
			Permission:    ptr(all()),
			Authorization: ssh.MarshalAuthorizedKey(key),
		}.EnsureDefaults()

		if _, err = s.Insert(g); err != nil {
			log.Println("failed to load:", g.Fingerprint, err)
			continue
		}
	}

	return nil
}
