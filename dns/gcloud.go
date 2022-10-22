package dns

import (
	"net/http"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/james-lawrence/bw/agent"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gdns "google.golang.org/api/dns/v1"
)

// GCloudDNSOption options for google cloud dns
type GCloudDNSOption func(*GoogleCloudDNS)

// GCloudDNSOptionCommon set common options for dns
func GCloudDNSOptionCommon(options ...Option) GCloudDNSOption {
	return func(r *GoogleCloudDNS) {
		r.config = r.config.merge(options...)
	}
}

// NewGoogleCloudDNSFromMetadata loads dns details from metadata server.
func NewGoogleCloudDNSFromMetadata(zoneID string, options ...GCloudDNSOption) (r GoogleCloudDNS, err error) {
	var (
		projectID, localZoneID string
	)

	if projectID, err = metadata.ProjectID(); err != nil {
		return r, errors.Wrap(err, "failed to lookup project ID")
	}

	return NewGoogleCloudDNS(projectID, localZoneID, options...), nil
}

// NewGoogleCloudDNS ...
func NewGoogleCloudDNS(projectID, zoneID string, options ...GCloudDNSOption) (r GoogleCloudDNS) {
	r = GoogleCloudDNS{
		projectID: projectID,
		zoneID:    zoneID,
	}

	for _, opt := range options {
		opt(&r)
	}

	return r
}

// GoogleCloudDNS ...
type GoogleCloudDNS struct {
	config
	projectID string
	zoneID    string
}

// Sample - samples the cluster and updates a dns entry.
func (t GoogleCloudDNS) Sample(c cluster) (err error) {
	var (
		client *http.Client
		s      *gdns.Service
		rr     *gdns.ResourceRecordSetsListResponse
	)

	if client, err = google.DefaultClient(oauth2.NoContext, gdns.CloudPlatformScope); err != nil {
		return errors.Wrap(err, "failed to build google cloud http client")
	}

	if s, err = gdns.New(client); err != nil {
		return errors.Wrap(err, "failed to build google dns service")
	}

	l := s.ResourceRecordSets.List(t.projectID, t.zoneID).Type("A")
	l = l.Name(t.config.FQDN)

	if rr, err = l.Do(); err != nil {
		return errors.Wrap(err, "failed to retrieve existing record")
	}

	sample := agent.NodesToPeers(c.GetN(t.MaximumNodes, []byte(t.config.FQDN))...)
	change := &gdns.Change{
		Additions: t.convert(t.config.peersToBind(sample...)...),
		Deletions: rr.Rrsets,
	}

	if change, err = s.Changes.Create(t.projectID, t.zoneID, change).Do(); err != nil {
		return errors.Wrap(err, "failed to apply dns changes")
	}

	return t.waitForSync(s, change)
}

func (t GoogleCloudDNS) waitForSync(s *gdns.Service, c *gdns.Change) (err error) {
	for {
		var (
			resp *gdns.Change
		)

		if resp, err = s.Changes.Get(t.projectID, t.zoneID, c.Id).Do(); err != nil {
			return errors.WithStack(err)
		}

		switch resp.Status {
		case "pending":
		case "done":
			return nil
		default:
			return errors.New(resp.Status)
		}

		time.Sleep(1 * time.Second)
	}
}

// Convert will convert a set of DNS records into a route53 ResourceRecord.
func (t GoogleCloudDNS) convert(records ...dns.A) []*gdns.ResourceRecordSet {
	additions := make([]string, 0, len(records))

	for _, record := range records {
		additions = append(additions, record.A.String())
	}

	return []*gdns.ResourceRecordSet{
		{
			Name:    t.config.FQDN,
			Type:    dns.TypeToString[dns.TypeA],
			Ttl:     int64(t.config.TTL),
			Rrdatas: additions,
		},
	}
}
