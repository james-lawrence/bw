package notary

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc/metadata"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/james-lawrence/bw/internal/x/sshx"
)

const (
	mdkey = "authorization"
)

// ErrUnauthorizedKey used when the key isn't authorized by the cluster its trying to connect too.
type ErrUnauthorizedKey struct{}

func (t ErrUnauthorizedKey) Error() string {
	path := bw.DefaultUserDirLocation(bw.DefaultNotaryKey) + ".pub"
	return fmt.Sprintf(`your key is unauthorized and will need to be added by an authorized user.
please give the following file to an authorized user "%s".
they can add the key using the command "bw notary insert %s"`, path, path)
}

// Format override standard error formatting.
func (t ErrUnauthorizedKey) Format(s fmt.State, verb rune) {
	io.WriteString(s, t.Error())
}

type keyGen func() ([]byte, error)

// NewAutoSigner - loads or generates a ssh key to sign RPC requests with.
// this method is only for use by clients and the new key will need to be added to the cluster.
func NewAutoSigner(comment string) (s Signer, err error) {
	return newAutoSignerPath(bw.DefaultUserDirLocation(bw.DefaultNotaryKey), comment, sshx.Auto)
}

// AutoSignerInfo returns the fingerprint and authorized ssh key.
func AutoSignerInfo() (fp string, pub []byte, err error) {
	var (
		pubk    ssh.PublicKey
		encoded []byte
	)

	location := bw.DefaultUserDirLocation(bw.DefaultNotaryKey) + ".pub"
	if encoded, err = ioutil.ReadFile(location); err != nil {
		return fp, pub, err
	}

	if pubk, _, _, _, err = ssh.ParseAuthorizedKey(encoded); err != nil {
		return fp, pub, err
	}

	return genFingerprint(ssh.MarshalAuthorizedKey(pubk)), encoded, nil
}

func newAutoSignerPath(location string, comment string, kgen keyGen) (s Signer, err error) {
	var (
		encoded    []byte
		pubencoded []byte
		pub        = location + ".pub"
	)

	log.Println("authorization key location", location)
	if encoded, err = ioutil.ReadFile(location); err == nil {
		return NewSigner(encoded)
	}

	// if we failed to read the file, check if the file exists.
	// if it does then something bad happened and we return an error.
	if _, serr := os.Stat(location); serr != nil && !os.IsNotExist(serr) {
		return s, errors.Wrapf(err, "unable to read authorization key: %s", location)
	}

	// if the file just didn't exist then great, lets generate it.
	log.Println("authorization key not found, generating automatically")

	if encoded, err = kgen(); err != nil {
		return s, errors.Wrap(err, "failed to generate authorization key")
	}

	if err = ioutil.WriteFile(location, encoded, 0600); err != nil {
		return s, errors.Wrap(err, "failed to generate authorization key")
	}

	if pubencoded, err = sshx.PublicKey(encoded); err != nil {
		return s, errors.Wrap(err, "failed to generate authorization key")
	}

	if strings.TrimSpace(comment) != "" {
		comment = " " + comment + "\r\n"
		pubencoded = append(bytes.TrimSpace(pubencoded), []byte(comment)...)
	}

	if err = ioutil.WriteFile(pub, pubencoded, 0600); err != nil {
		return s, errors.Wrap(err, "failed to generate authorization key")
	}

	return NewSigner(encoded)
}

// NewSigner a request signer from a private key.
func NewSigner(pkey []byte) (s Signer, err error) {
	var (
		pubkey []byte
		ss     ssh.Signer
	)

	if ss, err = ssh.ParsePrivateKey(pkey); err != nil {
		return s, err
	}

	if pubkey, err = sshx.PublicKey(pkey); err != nil {
		return s, err
	}

	return Signer{
		fingerprint: genFingerprint(pubkey),
		signer:      ss,
	}, nil
}

// Signer implements grpc's credentials.PerRPCCredentials
type Signer struct {
	fingerprint string
	signer      ssh.Signer
}

// GetRequestMetadata inserts authentication metadata into request.
func (t Signer) GetRequestMetadata(ctx context.Context, uri ...string) (m map[string]string, err error) {
	var (
		encoded string
		sig     Signature
	)
	tok := GenerateToken(t.fingerprint)

	if sig, err = genSignature(t.signer, tok); err != nil {
		return m, err
	}

	a := Authorization{
		Token:     &tok,
		Signature: &sig,
	}

	if encoded, err = EncodeAuthorization(a); err != nil {
		return m, err
	}

	return map[string]string{
		mdkey: encoded,
	}, nil
}

// RequireTransportSecurity indicates whether the credentials requires
// transport security.
func (t Signer) RequireTransportSecurity() bool {
	return false
}

// NewAuth authorization
func newAuth(s storage) auth {
	roots, err := loadAuthorizedKeys()
	logx.MaybeLog(errors.Wrap(err, "failed to load root credentials, may not be able to run notary service"))

	return auth{
		storage: s,
		roots:   roots,
	}
}

// Auth returns authorization.
type auth struct {
	storage storage
	roots   map[string]Grant
}

func (t auth) lookup(fingerprint string) (g Grant, ok bool) {
	var (
		err error
	)

	if g, ok = t.roots[fingerprint]; ok {
		return g, ok
	}

	if g, err = t.storage.Lookup(fingerprint); err == nil {
		return g, true
	}

	return g, false
}

// Authorize the given request context.
func (t auth) Authorize(ctx context.Context) Permission {
	var (
		err     error
		ok      bool
		md      metadata.MD
		a       Authorization
		g       Grant
		encoded []byte
		vals    []string
		pkey    ssh.PublicKey
	)

	if md, ok = metadata.FromIncomingContext(ctx); !ok {
		return none()
	}

	if vals = md.Get(mdkey); len(vals) != 1 {
		log.Println("recieved invalid token")
		return none()
	}

	if a, err = DecodeAuthorization(vals[0]); err != nil {
		log.Println(errors.Wrap(err, "failed to decode authorization"))
		return none()
	}

	if a.Token == nil || a.Signature == nil {
		log.Println(errors.Wrap(err, "failed to decode authorization"))
		return none()
	}

	if time.Now().UTC().Unix() > a.Token.Expires {
		log.Println("request token is expired")
		return none()
	}

	if g, ok = t.lookup(a.Token.Fingerprint); !ok {
		log.Println("no authorization found", a.Token.Fingerprint)
		return none()
	}

	if pkey, _, _, _, err = ssh.ParseAuthorizedKey(g.Authorization); err != nil {
		log.Println("parse key failed", a.Token.Fingerprint, len(g.Authorization), err)
		return none()
	}

	if encoded, err = genSignatureData(a.Token); err != nil {
		log.Println(errors.Wrap(err, "failed to generate signature data"))
		return none()
	}

	if err = pkey.Verify(encoded, a.Signature.sig()); err != nil {
		log.Println("verify request failed", a.Token.Fingerprint, err)
		return none()
	}

	return unwrap(g.Permission)
}

// GenerateToken generates a request token for the given fingerprint.
// this token is unsigned.
func GenerateToken(fingerprint string) (t Token) {
	ts := time.Now().UTC()
	return Token{
		ID:          uuid.Must(uuid.NewV4()).Bytes(),
		Fingerprint: fingerprint,
		Issued:      ts.Unix(),
		Expires:     ts.Add(10 * time.Second).Unix(),
	}
}

// EncodeAuthorization encodes an authorization into a b64 string.
func EncodeAuthorization(a Authorization) (encoded string, err error) {
	var (
		b []byte
	)

	if b, err = proto.Marshal(&a); err != nil {
		return "", errors.Wrap(err, "failed to generate authorization")
	}

	return base64.URLEncoding.EncodeToString(b), nil
}

// DecodeAuthorization decodes authorization into its token and signature.
func DecodeAuthorization(encoded string) (a Authorization, err error) {
	var (
		b64 []byte
	)

	if b64, err = base64.URLEncoding.DecodeString(encoded); err != nil {
		return a, err
	}

	if err = proto.Unmarshal(b64, &a); err != nil {
		return a, err
	}

	return a, nil
}

func loadAuthorizedKeys() (roots map[string]Grant, err error) {
	var (
		encoded []byte
		u       *user.User
	)
	roots = map[string]Grant{}

	if u, err = user.Current(); err != nil {
		return roots, err
	}

	authorizedKeysPath := filepath.Join(u.HomeDir, ".ssh", "authorized_keys")

	if encoded, err = ioutil.ReadFile(authorizedKeysPath); err != nil {
		return roots, err
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
		log.Println("loaded", g.Fingerprint)
		roots[g.Fingerprint] = g
	}

	log.Println("loaded", len(roots), "key(s)", authorizedKeysPath)

	return roots, nil
}
