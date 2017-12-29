package dns

import (
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/davecgh/go-spew/spew"
	clusterp "github.com/james-lawrence/bw/cluster"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

// Route53Option options for route53
type Route53Option func(*Route53)

// Route53OptionCommon set common options for route53
func Route53OptionCommon(options ...Option) Route53Option {
	return func(r *Route53) {
		r.config = r.config.merge(options...)
	}
}

// NewRoute53 new dns manager that keeps a sampling of dns records.
func NewRoute53(hostedZoneID string, region string, options ...Route53Option) (r Route53, err error) {
	var (
		sess *session.Session
		resp *route53.HostedZone
	)

	if sess, err = session.NewSession(); err != nil {
		return r, errors.WithStack(err)
	}

	svc := route53.New(sess, aws.NewConfig().WithRegion(region))

	r = Route53{
		hostedZoneID: hostedZoneID,
		svc:          svc,
	}

	for _, opt := range options {
		opt(&r)
	}

	log.Println("resolving zone id", r.hostedZoneID)
	if resp, err = r.lookupZone(r.hostedZoneID); err != nil {
		return r, err
	}

	r.hostedZoneID = aws.StringValue(resp.Id)
	return r, nil
}

// Route53 ...
type Route53 struct {
	config
	hostedZoneID string
	svc          *route53.Route53
}

// Sample - samples the cluster and updates a dns entry in route53
func (t Route53) Sample(c cluster) (err error) {
	var (
		r *route53.ChangeResourceRecordSetsOutput
	)

	sample := clusterp.NodesToPeers(c.GetN(t.MaximumNodes, []byte(t.config.FQDN))...)
	rrset := t.convertBindToRR(t.config.peersToBind(sample...)...)

	cb := route53.ChangeBatch{
		Changes: []*route53.Change{
			&route53.Change{
				Action: aws.String(route53.ChangeActionUpsert),
				ResourceRecordSet: &route53.ResourceRecordSet{
					Type:            aws.String(route53.RRTypeA),
					Name:            aws.String(t.config.FQDN),
					ResourceRecords: rrset,
					TTL:             aws.Int64(int64(t.config.TTL)),
				},
			},
		},
	}

	log.Println("updating dns", t.hostedZoneID, t.FQDN, spew.Sdump(sample))
	if r, err = t.svc.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{HostedZoneId: aws.String(t.hostedZoneID), ChangeBatch: &cb}); err != nil {
		return errors.Wrap(err, "failed to change record set")
	}

	log.Println("waiting for changes to propagate")
	return t.waitForSync(r.ChangeInfo)
}

var reAWSZoneID = regexp.MustCompile("^(/hostedzone/)?Z[A-Z0-9]{10,}$")

func (Route53) isZoneID(s string) bool {
	return reAWSZoneID.MatchString(s)
}

var unescaper = strings.NewReplacer(`\057`, "/", `\052`, "*")

func (Route53) zoneName(s string) string {
	return unescaper.Replace(strings.TrimRight(s, "."))
}

func (t Route53) lookupZone(nameOrID string) (*route53.HostedZone, error) {
	if t.isZoneID(nameOrID) {
		// lookup by id
		id := nameOrID
		if !strings.HasPrefix(nameOrID, "/hostedzone/") {
			id = "/hostedzone/" + id
		}
		req := route53.GetHostedZoneInput{
			Id: aws.String(id),
		}
		resp, err := t.svc.GetHostedZone(&req)
		if err, ok := err.(awserr.Error); ok && err.Code() == route53.ErrCodeNoSuchHostedZone {
			return nil, errors.Wrapf(err, "zone '%s' not found", nameOrID)
		}
		return resp.HostedZone, errors.WithStack(err)
	}

	// lookup by name
	matches := []route53.HostedZone{}
	req := route53.ListHostedZonesByNameInput{
		DNSName: aws.String(nameOrID),
	}

	resp, err := t.svc.ListHostedZonesByName(&req)
	for _, zone := range resp.HostedZones {
		if t.zoneName(*zone.Name) == t.zoneName(nameOrID) {
			matches = append(matches, *zone)
		}
	}
	switch len(matches) {
	case 0:
		return nil, errors.Wrapf(err, "zone '%s' not found", nameOrID)
	case 1:
		return &matches[0], nil
	default:
		return nil, errors.Wrapf(err, "multiple zones match '%s' you will need to use Zone ID to uniquely identify the zone", nameOrID)
	}
}

func (t Route53) waitForSync(change *route53.ChangeInfo) (err error) {
	var (
		resp *route53.GetChangeOutput
	)

	for {
		req := route53.GetChangeInput{Id: change.Id}
		if resp, err = t.svc.GetChange(&req); err != nil {
			return errors.WithStack(err)
		}

		switch status := *resp.ChangeInfo.Status; status {
		case route53.ChangeStatusPending:
		case route53.ChangeStatusInsync:
			return nil
		default:
			return errors.New(status)
		}

		time.Sleep(1 * time.Second)
	}
}

// ConvertBindToRR will convert a DNS record into a route53 ResourceRecord.
func (t Route53) convertBindToRR(records ...dns.A) []*route53.ResourceRecord {
	flattenedIP := make([]*route53.ResourceRecord, 0, len(records))
	for _, record := range records {
		flattenedIP = append(flattenedIP, &route53.ResourceRecord{
			Value: aws.String(record.A.String()),
		})
	}

	return flattenedIP
}
