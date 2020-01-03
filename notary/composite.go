package notary

// NewComposite storage.
func NewComposite(p storage, buckets ...storage) Composite {
	return Composite{
		primary: p,
		buckets: buckets,
	}
}

// Composite combine multiple storages into a single api.
type Composite struct {
	primary storage   // the mutatable bucket, allows for insertions.
	buckets []storage // read only buckets. cannot be mutated by the composite.
}

// Lookup scan each bucket for the fingerprint starting with the primary.
// returns the last error encountered.
func (t Composite) Lookup(fingerprint string) (g Grant, err error) {
	if g, err = t.primary.Lookup(fingerprint); err == nil {
		return g, err
	}

	for _, b := range t.buckets {
		if g, err = b.Lookup(fingerprint); err == nil {
			return g, err
		}
	}

	return g, err
}

// Insert the grant into the primary.
func (t Composite) Insert(g Grant) (Grant, error) {
	return t.primary.Insert(g)
}

// Delete the grant from the primary.
func (t Composite) Delete(g Grant) (Grant, error) {
	return t.primary.Delete(g)
}
