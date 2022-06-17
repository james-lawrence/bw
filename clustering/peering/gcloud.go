package peering

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/x/envx"
	"github.com/james-lawrence/bw/internal/x/errorsx"
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

	if r := pathsuffix(createdBy, "/"); r == "" {
		return results, errors.New("invalid created by")
	} else {
		createdBy = r
	}

	if envx.Boolean(false, bw.EnvLogsVerbose) {
		log.Println("instance created by", createdBy)
	}

	if peers, cause := t.standard(ctx, c, project, zone, createdBy); cause == nil {
		results = append(results, peers...)
	} else {
		if envx.Boolean(false, bw.EnvLogsVerbose) {
			log.Println("instance group peers failed", cause)
		}
		err = errorsx.Compact(err, cause)
	}

	if peers, cause := t.region(ctx, c, project, zone, createdBy); cause == nil {
		results = append(results, peers...)
	} else {
		if envx.Boolean(false, bw.EnvLogsVerbose) {
			log.Println("region instance group peers failed", cause)
		}
		err = errorsx.Compact(err, cause)
	}

	if len(results) > 0 {
		err = nil
	}

	return results, err
}

func (t GCloudTargetPool) region(ctx context.Context, c *compute.Service, project string, zone string, createdBy string) (results []string, err error) {
	var (
		region string
		igreq  *compute.RegionInstanceGroupManagersGetCall
		igresp *compute.InstanceGroupManager
		req    *compute.RegionInstanceGroupManagersListManagedInstancesCall
		resp   *compute.RegionInstanceGroupManagersListInstancesResponse
	)
	if prefix := regionstring(zone, "-"); prefix == "" {
		log.Println("cannot convert zone to region", zone)
		return results, nil
	} else {
		region = prefix
	}
	igreq = c.RegionInstanceGroupManagers.Get(project, region, createdBy)
	if igresp, err = igreq.Context(ctx).Do(); err != nil {
		return results, errors.WithStack(err)
	}

	zones := extractzones(igresp.DistributionPolicy)
	req = c.RegionInstanceGroupManagers.ListManagedInstances(project, region, createdBy)

	if resp, err = req.Context(ctx).Do(); err != nil {
		return results, errors.WithStack(err)
	}

	maximum := 2 * t.Maximum
	if maximum == 0 {
		maximum = len(resp.ManagedInstances)
	}

	if envx.Boolean(false, bw.EnvLogsVerbose) {
		log.Println("located gcloud instance group manager")
	}

	for _, inst := range resp.ManagedInstances {
		if ip := t.ip(c, project, inst, zones...); len(ip) > 0 {
			results = append(results, fmt.Sprintf("%s:%d", ip, t.Port))

			// 2 * maximum in case some are in a building state and to handle ip == current instance
			if len(results) > maximum {
				return results, nil
			}
		}
	}
	return results, nil
}

func (t GCloudTargetPool) standard(ctx context.Context, c *compute.Service, project string, zone string, createdBy string) (results []string, err error) {
	var (
		req  *compute.InstanceGroupManagersListManagedInstancesCall
		resp *compute.InstanceGroupManagersListManagedInstancesResponse
	)
	req = c.InstanceGroupManagers.ListManagedInstances(project, zone, createdBy)

	if resp, err = req.Context(ctx).Do(); err != nil {
		return results, errors.WithStack(err)
	}

	maximum := 2 * t.Maximum
	if maximum == 0 {
		maximum = len(resp.ManagedInstances)
	}

	if envx.Boolean(false, bw.EnvLogsVerbose) {
		log.Println("located gcloud instance group manager")
	}

	for _, inst := range resp.ManagedInstances {
		if ip := t.ip(c, project, inst, zone); len(ip) > 0 {
			results = append(results, fmt.Sprintf("%s:%d", ip, t.Port))

			// 2 * maximum in case some are in a building state and to handle ip == current instance
			if len(results) > maximum {
				return results, nil
			}
		}
	}

	return results, nil
}

func (t GCloudTargetPool) ip(c *compute.Service, project string, mi *compute.ManagedInstance, zones ...string) string {
	var (
		err error
	)

	for _, zone := range zones {
		var (
			cause    error
			instance *compute.Instance
		)

		if instance, cause = c.Instances.Get(project, zone, strconv.FormatUint(mi.Id, 10)).Do(); cause != nil {
			err = errorsx.Compact(err, cause)
			continue
		}

		if envx.Boolean(false, bw.EnvLogsVerbose) {
			log.Println("gcloud peer info", spew.Sdump(instance))
		}

		// return first IP found.
		for _, n := range instance.NetworkInterfaces {
			return n.NetworkIP
		}
	}

	if err != nil {
		log.Println("failed to retrieve instance", strconv.FormatUint(mi.Id, 10), err)
	}

	return ""
}

func extractzones(policy *compute.DistributionPolicy) (res []string) {
	if policy == nil {
		return res
	}

	for _, z := range policy.Zones {
		res = append(res, pathsuffix(z.Zone, "/"))
	}

	return res
}

func pathsuffix(s string, sep string) string {
	var (
		idx int
	)

	if idx = strings.LastIndex(s, sep); idx < 0 || idx > len(s)-1 {
		return ""
	}

	return s[idx+1:]
}

func regionstring(s string, sep string) string {
	var (
		idx int
	)

	if idx = strings.LastIndex(s, sep); idx < 0 || idx > len(s)-1 {
		return ""
	}

	return s[:idx]
}
