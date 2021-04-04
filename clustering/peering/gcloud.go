package peering

import (
	"context"
	"log"
	"strconv"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"github.com/pkg/errors"
	"google.golang.org/api/compute/v1"
)

// GCloudTargetPool based peering
type GCloudTargetPool struct {
	Port    int // port to connect to.
	Maximum int
}

// Peers - reads peers from aws Autoscaling groups.
func (t GCloudTargetPool) Peers(ctx context.Context) (results []string, err error) {
	var (
		c         *compute.Service
		project   string
		zone      string
		createdBy string
		idx       int
		req       *compute.InstanceGroupManagersListManagedInstancesCall
		resp      *compute.InstanceGroupManagersListManagedInstancesResponse
	)

	if !metadata.OnGCE() {
		return results, nil
	}

	if project, err = metadata.ProjectID(); err != nil {
		return results, errors.WithStack(err)
	}

	if zone, err = metadata.Zone(); err != nil {
		return results, errors.WithStack(err)
	}

	if c, err = compute.NewService(ctx); err != nil {
		return results, errors.WithStack(err)
	}

	if createdBy, err = metadata.InstanceAttributeValue("created-by"); err != nil {
		return results, errors.WithStack(err)
	}

	if idx = strings.LastIndex(createdBy, "/"); idx < 0 || idx > len(createdBy)-1 {
		return results, errors.New("invalid created by")
	}
	createdBy = createdBy[idx+1:]

	req = c.InstanceGroupManagers.ListManagedInstances(project, zone, createdBy)

	if resp, err = req.Do(); err != nil {
		return results, errors.WithStack(err)
	}

	maximum := 2 * t.Maximum
	if maximum == 0 {
		maximum = len(resp.ManagedInstances)
	}

	for _, inst := range resp.ManagedInstances {
		if ip := t.ip(c, project, zone, inst); len(ip) > 0 {
			results = append(results, ip)

			// 2 * maximum in case some are in a building state and to handle ip == current instance
			if len(results) > maximum {
				return results, nil
			}
		}
	}

	return results, nil
}

func (t GCloudTargetPool) ip(c *compute.Service, project, zone string, mi *compute.ManagedInstance) string {
	var (
		err      error
		id       string
		instance *compute.Instance
	)

	if instance, err = c.Instances.Get(project, zone, strconv.FormatUint(mi.Id, 10)).Do(); err != nil {
		log.Println("failed to retrieve instance", strconv.FormatUint(mi.Id, 10), id, err)
		return ""
	}

	// return first IP found.
	for _, n := range instance.NetworkInterfaces {
		return n.NetworkIP
	}

	// log.Println("response", spew.Sdump(instance))
	return ""
}
