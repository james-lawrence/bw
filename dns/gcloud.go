package dns

import (
	"net/http"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	dns "google.golang.org/api/dns/v1"
)

// GCloudDNSOption options for google cloud dns
type GCloudDNSOption func(*GoogleCloudDNS)

// NewGoogleCloudDNS ...
func NewGoogleCloudDNS(hostedZoneID string, region string, options ...GCloudDNSOption) (r GoogleCloudDNS, err error) {
	return GoogleCloudDNS{}, nil
}

// GoogleCloudDNS ...
type GoogleCloudDNS struct{}

// Sample - samples the cluster and updates a dns entry.
func (t GoogleCloudDNS) Sample(c cluster) (err error) {
	const (
		scope = "https://www.googleapis.com/auth/ndev.clouddns.readwrite"
	)
	var (
		client *http.Client
		s      *dns.Service
	)

	if client, err = google.DefaultClient(oauth2.NoContext); err != nil {
		return errors.Wrap(err, "failed to build google cloud http client")
	}

	if s, err = dns.New(client); err != nil {
		return errors.Wrap(err, "failed to build google dns service")
	}

	_ = s.ChangesService.Create()

	return nil
}
