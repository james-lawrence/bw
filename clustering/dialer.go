package clustering

import (
	"github.com/pkg/errors"
)

// DialOption ...
type DialOption func(*Dialer)

// NewDialer creates a dialer to connect to the cluster.
func NewDialer(doptions ...Option) Dialer {
	return Dialer{
		dOptions: doptions,
	}
}

// Dialer used to join a cluster.
type Dialer struct {
	dOptions []Option // default Options
}

// Dial ...
func (t Dialer) Dial(options ...Option) (Cluster, error) {
	options = append(t.dOptions, options...)
	c, err := NewOptions(options...).NewCluster()
	return c, errors.Wrap(err, "failed to join cluster")
}
