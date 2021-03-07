package notary

// EnsureDefaults for the current grant.
func (t Grant) EnsureDefaults() *Grant {
	if t.Permission == nil {
		t.Permission = none()
	}

	t.Fingerprint = genFingerprint(t.Authorization)

	return &t
}
