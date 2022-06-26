package gcloud

import (
	"context"
	"log"

	"cloud.google.com/go/compute/metadata"
	"github.com/james-lawrence/bw/internal/gcloudx"
	"github.com/pkg/errors"
	"google.golang.org/api/compute/v1"
)

// TargetPoolDetach detach an instance from its target pool.
func TargetPoolDetach(ctx context.Context) (err error) {
	var (
		tpools []string
	)

	if tpools, err = targetPools(ctx); err != nil {
		return err
	}

	for _, uri := range tpools {
		log.Println("target pool", uri)
	}

	return nil
}

// TargetPoolAttach attach an instance to its target pool.
func TargetPoolAttach(ctx context.Context) (err error) {
	var (
		tpools []string
	)

	if tpools, err = targetPools(ctx); err != nil {
		return err
	}

	for _, uri := range tpools {
		log.Println("target pool", uri)
	}

	return nil
}

func targetPools(ctx context.Context) (_ []string, err error) {
	var (
		c           *compute.Service
		project     string
		zone        string
		createdBy   string
		tmp         []string
		targetPools []string
	)

	if !metadata.OnGCE() {
		return targetPools, errors.Errorf("unable to detach from target pool: requires running within a gce environment")
	}

	if project, err = metadata.ProjectID(); err != nil {
		return targetPools, errors.WithStack(err)
	}

	if zone, err = metadata.Zone(); err != nil {
		return targetPools, errors.WithStack(err)
	}

	if c, err = compute.NewService(ctx); err != nil {
		return targetPools, errors.WithStack(err)
	}

	if createdBy, err = gcloudx.InstanceGroupManagerName(); err != nil {
		return targetPools, err
	}

	log.Println("instance manager", createdBy)

	if tmp, err = igmTargetPools(ctx, c, project, zone, createdBy); err != nil {
		return targetPools, err
	} else {
		targetPools = append(targetPools, tmp...)
	}

	if tmp, err = rigmTargetPools(ctx, c, project, zone, createdBy); err != nil {
		return targetPools, err
	} else {
		targetPools = append(targetPools, tmp...)
	}

	return targetPools, nil
}

func igmTargetPools(ctx context.Context, c *compute.Service, project, zone, createdBy string) (_ []string, err error) {
	var (
		igm *compute.InstanceGroupManager
	)

	if igm, err = c.InstanceGroupManagers.Get(project, zone, createdBy).Context(ctx).Do(); err != nil {
		return []string(nil), errors.WithStack(err)
	}

	return igm.TargetPools, nil
}

func rigmTargetPools(ctx context.Context, c *compute.Service, project, zone, createdBy string) (_ []string, err error) {
	var (
		region string
		igm    *compute.InstanceGroupManager
	)

	if region = gcloudx.ZonalRegion(zone, "-"); region == "" {
		log.Println("cannot convert zone to region", zone)
		return []string(nil), nil
	}

	igm, err = c.RegionInstanceGroupManagers.Get(project, region, createdBy).Context(ctx).Do()
	if err != nil {
		return []string(nil), errors.WithStack(err)
	}

	return igm.TargetPools, nil
}
