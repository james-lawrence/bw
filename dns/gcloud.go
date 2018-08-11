package dns

import (
	"net/http"

	"cloud.google.com/go/compute/metadata"
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
		client      *http.Client
		s           *dns.Service
		projectID   string
		localZoneID string
	)

	if projectID, err = metadata.ProjectID(); err != nil {
		return errors.Wrap(err, "failed to lookup project ID")
	}

	if localZoneID, err = metadata.Zone(); err != nil {
		return errors.Wrap(err, "failed to lookup local zone ID")
	}

	if client, err = google.DefaultClient(oauth2.NoContext, scope); err != nil {
		return errors.Wrap(err, "failed to build google cloud http client")
	}

	if s, err = dns.New(client); err != nil {
		return errors.Wrap(err, "failed to build google dns service")
	}

	change := dns.Change{}

	// TODO: update records in dns.
	if _, err = s.Changes.Create(projectID, localZoneID, &change).Do(); err != nil {
		return errors.Wrap(err, "failed to apply dns changes")
	}

	return nil
}
