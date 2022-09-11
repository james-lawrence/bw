package notary

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/debugx"
	"github.com/james-lawrence/bw/internal/sshx"
	"github.com/james-lawrence/bw/internal/systemx"
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

	if bin, err = io.ReadAll(in); err != nil {
		return c, err
	}

	if err = bw.ExpandAndDecode(bin, &nc); err != nil {
		return c, err
	}

	authority := append(
		nc.Config.Authority,
		filepath.Join(root, bw.AuthKeysFile),
	)
	buckets := make([]storage, 0, len(authority))

	log.Println("authorizations load initiated", len(authority))
	defer log.Println("authorizations load completed", len(authority))
	for _, p := range authority {
		var (
			b storage
		)

		if b, err = newFile(p); err != nil {
			return c, errors.Wrapf(err, "failed to initialize: %s", p)
		}

		buckets = append(buckets, b)
	}

	return NewComposite(root, directory, buckets...), nil
}

// CloneAuthorizationFile copies the authorization from one location to another.
func CloneAuthorizationFile(from, to string) (err error) {
	var (
		ff *os.File
		tf *os.File
	)

	debugx.Println("synchronizing authorization", from, "->", to)
	if ff, err = os.Open(from); err != nil {
		return err
	}
	defer ff.Close()

	if tf, err = os.CreateTemp(filepath.Dir(to), fmt.Sprintf("%s.sync", filepath.Base(to))); err != nil {
		return err
	}
	defer tf.Close()

	if err = os.Chmod(tf.Name(), 0600); err != nil {
		return err
	}

	if _, err = io.Copy(tf, ff); err != nil {
		return err
	}

	debugx.Println("renaming", tf.Name(), "->", to)
	return os.Rename(tf.Name(), to)
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

func loadAuthorizedKeys(s storage, path string) (err error) {
	var (
		encoded []byte
	)

	if !systemx.FileExists(path) {
		log.Println("not loading", path, "does not exist")
		return nil
	}

	if encoded, err = os.ReadFile(path); err != nil {
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

		g := (&Grant{
			Permission:    UserFull(),
			Authorization: ssh.MarshalAuthorizedKey(key),
		}).EnsureDefaults()

		if _, err = s.Insert(g); err != nil {
			log.Println("failed to load:", g.Fingerprint, err)
			continue
		}
	}

	return nil
}

// ReplaceAuthorizedKey replace an authorized
func ReplaceAuthorizedKey(path, fingerprint string, rpub []byte) (err error) {
	var (
		auths   *os.File
		buf     *os.File
		encoded []byte
	)

	if auths, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600); err != nil {
		return errors.WithStack(err)
	}
	defer auths.Close()

	if buf, err = os.CreateTemp(filepath.Dir(path), fmt.Sprintf("%s.*", filepath.Base(path))); err != nil {
		return errors.Wrap(err, "unable to open buffer")
	}
	defer os.Remove(buf.Name())
	defer buf.Close()

	debugx.Println("replacing authorization key within", path)

	if encoded, err = os.ReadFile(path); err != nil {
		return err
	}

	for len(encoded) != 0 {
		var (
			pubencoded []byte
			comment    string
			key        ssh.PublicKey
		)

		if key, comment, _, encoded, err = ssh.ParseAuthorizedKey(encoded); err != nil {
			if sshx.IsNoKeyFound(err) {
				continue
			}
			log.Println(err)
			continue
		}

		pubencoded = ssh.MarshalAuthorizedKey(key)

		if sshx.FingerprintSHA256(pubencoded) == fingerprint {
			continue
		}

		if _, err = buf.Write(sshx.Comment(pubencoded, comment)); err != nil {
			return err
		}
	}

	if _, err = buf.Write(rpub); err != nil {
		return err
	}

	return CloneAuthorizationFile(buf.Name(), path)
}
