package peering

import (
	"context"
	"net"
	"strconv"

	"github.com/james-lawrence/bw/awsx"
)

// AWSAutoscaling based peering
type AWSAutoscaling struct {
	Port               int      // port to connect to.
	SupplimentalGroups []string // additional autoscaling group names to check
}

// Peers - reads peers from aws Autoscaling groups.
func (t AWSAutoscaling) Peers(ctx context.Context) (results []string, err error) {
	instances, err := awsx.AutoscalingPeers(ctx, t.SupplimentalGroups...)
	if err != nil {
		return []string(nil), err
	}

	result := make([]string, 0, len(instances))
	for _, i := range instances {
		if i.PrivateIpAddress == nil {
			continue
		}

		result = append(result, net.JoinHostPort(*i.PrivateIpAddress, strconv.Itoa(t.Port)))
	}

	return result, nil
}
