package notary

import "github.com/james-lawrence/bw/internal/sshx"

// EnsureDefaults for the current grant.
func (t Grant) EnsureDefaults() *Grant {
	if t.Permission == nil {
		t.Permission = none()
	}

	if t.Fingerprint == "" {
		t.Fingerprint = sshx.FingerprintSHA256(t.Authorization)
	}

	return &t
}
