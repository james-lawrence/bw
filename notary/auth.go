package notary

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/james-lawrence/bw/internal/sshx"
)

const (
	mdkey = "authorization"
)

// ErrUnauthorizedKey used when the key isn't authorized by the cluster its trying to connect too.
type ErrUnauthorizedKey struct{}

func (t ErrUnauthorizedKey) Error() string {
	path := PublicKeyPath()
	return fmt.Sprintf(`your key is unauthorized and will need to be added by an authorized user.
please give the following file to an authorized user "%s".
they can add the key using the command "bw notary insert %s"`, path, path)
}

// Format override standard error formatting.
func (t ErrUnauthorizedKey) Format(s fmt.State, verb rune) {
	if _, err := io.WriteString(s, t.Error()); err != nil {
		log.Println("unable to format unauthorized error, ignored", err)
	}
}

type keyGen func() ([]byte, error)

// PublicKeyPath generates the path to the public key on disk for
// a client.
func PublicKeyPath() string {
	return bw.DefaultUserDirLocation(bw.DefaultNotaryKey) + ".pub"
}

// PrivateKeyPath generates the path to the private key on disk for
// a client.
func PrivateKeyPath() string {
	return bw.DefaultUserDirLocation(bw.DefaultNotaryKey)
}

// ClearAutoSignerKey clears the autosigner from disk.
func ClearAutoSignerKey() error {
	return errorsx.Compact(
		os.Remove(PrivateKeyPath()),
		os.Remove(PublicKeyPath()),
	)
}

// NewAgentSigner - loads or generates a ssh key to sign RPC requests with.
// this method is only for use by agents.
func NewAgentSigner(root string) (s Signer, err error) {
	return newAutoSignerPath(filepath.Join(root, bw.DefaultAgentNotaryKey), "", rsax.Auto)
}

// NewAutoSigner - loads or generates a ssh key to sign RPC requests with.
// this method is only for use by clients and the new key will need to be added to the cluster.
func NewAutoSigner(comment string) (s Signer, err error) {
	return newAutoSignerPath(bw.DefaultUserDirLocation(bw.DefaultNotaryKey), comment, rsax.Auto)
}

func NewDeterministicSigner(seed []byte, comment string) (s Signer, err error) {
	return newAutoSignerPath(bw.DefaultUserDirLocation(bw.DefaultNotaryKey), comment, rsax.AutoDeterministic(seed))
}

// NewPresharedKeySigner creates a signer using a preshared key for deterministic credential generation.
// This allows agents to automatically have valid credentials without manual provisioning.
func NewPresharedKeySigner(presharedKey, agentID, comment string) (s Signer, err error) {
	seed := sha256.Sum256([]byte(presharedKey + ":" + agentID))
	return newAutoSignerPath(bw.DefaultUserDirLocation(bw.DefaultNotaryKey), comment, rsax.AutoDeterministic(seed[:]))
}

// NewAgentPresharedKeySigner creates an agent signer using a preshared key.
func NewAgentPresharedKeySigner(root, presharedKey, agentID string) (s Signer, err error) {
	seed := sha256.Sum256([]byte(presharedKey + ":agent:" + agentID))
	return newAutoSignerPath(filepath.Join(root, bw.DefaultAgentNotaryKey), "", rsax.AutoDeterministic(seed[:]))
}

// GeneratePresharedKeyCredentials generates the public key for a given preshared key + agent ID combination.
// This is used to pre-populate the authorization system with valid credentials.
func GeneratePresharedKeyCredentials(presharedKey, agentID string) (fingerprint string, pubKey []byte, err error) {
	seed := sha256.Sum256([]byte(presharedKey + ":agent:" + agentID))

	privateKey, err := rsax.AutoDeterministic(seed[:])()
	if err != nil {
		return "", nil, err
	}

	pubKey, err = sshx.PublicKey(privateKey)
	if err != nil {
		return "", nil, err
	}

	fingerprint = sshx.FingerprintSHA256(pubKey)
	return fingerprint, pubKey, nil
}

// BootstrapPresharedKeyCredentials automatically adds credentials for all agents that would use the preshared key.
// This should be called on cluster startup if preshared keys are enabled.
func BootstrapPresharedKeyCredentials(storage storage, presharedKey string, agentIDs []string) error {
	for _, agentID := range agentIDs {
		fingerprint, pubKey, err := GeneratePresharedKeyCredentials(presharedKey, agentID)
		if err != nil {
			log.Printf("failed to generate preshared credentials for agent %s: %v", agentID, err)
			continue
		}

		grant := &Grant{
			Permission:    agent(),
			Authorization: pubKey,
			Fingerprint:   fingerprint,
		}

		if _, err := storage.Insert(grant); err != nil {
			log.Printf("failed to insert preshared credentials for agent %s: %v", agentID, err)
			continue
		}

		log.Printf("bootstrapped preshared credentials for agent %s (fingerprint: %s)", agentID, fingerprint)
	}

	return nil
}

// AutoSignerInfo returns the fingerprint and authorized ssh key.
func AutoSignerInfo() (fp string, pub []byte, err error) {
	var (
		pubk    ssh.PublicKey
		encoded []byte
	)

	if encoded, err = os.ReadFile(PublicKeyPath()); err != nil {
		return fp, pub, err
	}

	if pubk, _, _, _, err = ssh.ParseAuthorizedKey(encoded); err != nil {
		return fp, pub, err
	}

	return sshx.FingerprintSHA256(ssh.MarshalAuthorizedKey(pubk)), encoded, nil
}

func newAutoSignerPath(location string, comment string, kgen keyGen) (s Signer, err error) {
	var (
		encoded    []byte
		pubencoded []byte
		pub        = location + ".pub"
	)

	if envx.Boolean(false, bw.EnvLogsVerbose) {
		log.Println("authorization key location", location)
	}

	if encoded, err = os.ReadFile(location); err == nil {
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

	if err = os.MkdirAll(filepath.Dir(location), 0700); err != nil {
		return s, errors.Wrapf(err, "failed to create credential directory '%s'", filepath.Dir(location))
	}

	if err = os.WriteFile(location, encoded, 0600); err != nil {
		return s, errors.Wrapf(err, "failed to write authorization key '%s'", location)
	}

	if pubencoded, err = sshx.PublicKey(encoded); err != nil {
		return s, errors.Wrap(err, "failed to write public key")
	}

	if err = os.WriteFile(pub, sshx.Comment(pubencoded, comment), 0600); err != nil {
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
		fingerprint: sshx.FingerprintSHA256(pubkey),
		signer:      ss,
	}, nil
}

// Signer implements grpc's credentials.PerRPCCredentials
type Signer struct {
	fingerprint string
	signer      ssh.Signer
}

// AutoSignerInfo returns the fingerprint and authorized ssh key.
func (t Signer) AutoSignerInfo() (fp string, pub []byte, err error) {
	return t.fingerprint, ssh.MarshalAuthorizedKey(t.signer.PublicKey()), nil
}

func (t Signer) Token() (encoded string, err error) {
	var (
		sig *Signature
	)

	tok := GenerateToken(t.fingerprint)

	if sig, err = genTokenSignature(t.signer, &tok); err != nil {
		return "", err
	}

	a := Authorization{
		Token:     &tok,
		Signature: sig,
	}

	return EncodeAuthorization(&a)
}

// GetRequestMetadata inserts authentication metadata into request.
func (t Signer) GetRequestMetadata(ctx context.Context, uri ...string) (m map[string]string, err error) {
	var (
		encoded string
	)

	if encoded, err = t.Token(); err != nil {
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

func NewAuthChecker(s storage, check func(*Permission) error) AuthChecker {
	return AuthChecker{storage: s, check: check}
}

type AuthChecker struct {
	storage storage
	check   func(*Permission) error
}

func (t AuthChecker) Authorization(encoded []byte) (err error) {
	return t.check(decode(t.storage, string(encoded)))
}

func decode(s storage, encodedt string) *Permission {
	var (
		err     error
		encoded []byte
		a       *Authorization
		g       *Grant
		pkey    ssh.PublicKey
	)

	if a, err = DecodeAuthorization(encodedt); err != nil {
		log.Println(errors.Wrap(err, "failed to decode authorization"))
		return none()
	}

	if a.Token == nil || a.Signature == nil {
		log.Println(errors.Wrap(err, "missing token/signature"))
		return none()
	}

	if time.Now().UTC().Unix() > a.Token.Expires {
		log.Println("request token is expired")
		return none()
	}

	if g, err = s.Lookup(a.Token.Fingerprint); err != nil {
		log.Println(errors.Wrapf(err, "unknown authorization: %s", a.Token.Fingerprint))
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

	return g.Permission
}

// NewAuth authorization
func NewAuth(s storage) Auth {
	return newAuth(s)
}

func newAuth(s storage) Auth {
	return Auth{
		storage: s,
	}
}

// Auth returns authorization.
type Auth struct {
	storage storage
}

// Authorize the given request context.
func (t Auth) Authorize(ctx context.Context) *Permission {
	var (
		ok   bool
		md   metadata.MD
		vals []string
	)

	if md, ok = metadata.FromIncomingContext(ctx); !ok {
		log.Println("token metadata")
		return none()
	}

	if vals = md.Get(mdkey); len(vals) != 1 {
		log.Println("recieved invalid token", len(vals))
		return none()
	}

	return decode(t.storage, vals[0])
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
func EncodeAuthorization(a *Authorization) (encoded string, err error) {
	var (
		b []byte
	)

	if b, err = proto.Marshal(a); err != nil {
		return "", errors.Wrap(err, "failed to generate authorization")
	}

	return base64.URLEncoding.EncodeToString(b), nil
}

// DecodeAuthorization decodes authorization into its token and signature.
func DecodeAuthorization(encoded string) (_ *Authorization, err error) {
	var (
		a   Authorization
		b64 []byte
	)

	if b64, err = base64.URLEncoding.DecodeString(encoded); err != nil {
		return nil, err
	}

	if err = proto.Unmarshal(b64, &a); err != nil {
		return nil, err
	}

	return &a, nil
}

type auth interface {
	Authorize(ctx context.Context) *Permission
}

func NewAgentAuth(a auth) AgentAuth {
	return AgentAuth{
		auth: a,
	}
}

type AgentAuth struct {
	auth
}

func (t AgentAuth) Deploy(ctx context.Context) error {
	if t.Authorize(ctx).Deploy {
		return nil
	}

	return status.Error(codes.PermissionDenied, "invalid credentials")
}
