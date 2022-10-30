// Package gcloud utility functionality for deploying in gcp
// requires:
// - compute.targetPools.removeInstance
// - compute.targetPools.addInstance
// - compute.instances.use
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
	return modify(ctx, detach)
}

// TargetPoolAttach attach an instance to its target pool.
func TargetPoolAttach(ctx context.Context) (err error) {
	return modify(ctx, attach)
}

func detach(ctx context.Context, project string, region string, c *compute.Service, instLink string, tpools ...string) error {
	for _, uri := range tpools {
		log.Println("target pool", uri, instLink)
		op, err := c.TargetPools.RemoveInstance(
			project,
			region,
			gcloudx.PathSuffix(uri, "/"),
			&compute.TargetPoolsRemoveInstanceRequest{
				Instances: []*compute.InstanceReference{
					{Instance: instLink},
				},
			},
		).Context(ctx).Do()

		if err != nil {
			return err
		}

		log.Println("op", op.Name, op.Region, op.Status)
		for {
			op, err = c.RegionOperations.Wait(project, region, op.Name).Context(ctx).Do()
			if err != nil {
				return err
			}

			log.Println("op", op.Name, op.Region, op.Status)
			if op.Status == "DONE" {
				return nil
			}
		}
	}

	return nil
}

func attach(ctx context.Context, project string, region string, c *compute.Service, instLink string, tpools ...string) error {
	for _, uri := range tpools {
		log.Println("target pool", uri, instLink)
		op, err := c.TargetPools.AddInstance(
			project,
			region,
			gcloudx.PathSuffix(uri, "/"),
			&compute.TargetPoolsAddInstanceRequest{
				Instances: []*compute.InstanceReference{
					{Instance: instLink},
				},
			},
		).Context(ctx).Do()

		if err != nil {
			return err
		}

		log.Println("op", op.Name, op.Region, op.Status)
		for {
			op, err = c.RegionOperations.Wait(project, region, op.Name).Context(ctx).Do()
			if err != nil {
				return err
			}
			log.Println("op", op.Name, op.Region, op.Status)
			if op.Status == "DONE" {
				return nil
			}
		}
	}

	return nil
}

type op func(ctx context.Context, project string, region string, c *compute.Service, instLink string, tpools ...string) error

func modify(ctx context.Context, op op) (err error) {
	var (
		c        *compute.Service
		instance *compute.Instance
		id       string
		project  string
		region   string
		zone     string
		tpools   []string
	)

	if !metadata.OnGCE() {
		return errors.Errorf("unable to detach from target pool: requires running within a gce environment")
	}

	if project, err = metadata.ProjectID(); err != nil {
		return errors.WithStack(err)
	}

	if zone, err = metadata.Zone(); err != nil {
		return errors.WithStack(err)
	}

	if prefix := gcloudx.ZonalRegion(zone, "-"); prefix == "" {
		return errors.Errorf("cannot convert zone to region: %s", zone)
	} else {
		region = prefix
	}

	if c, err = compute.NewService(ctx); err != nil {
		return errors.WithStack(err)
	}

	if id, err = metadata.InstanceID(); err != nil {
		return err
	}

	if instance, err = c.Instances.Get(project, zone, id).Context(ctx).Do(); err != nil {
		return err
	}

	if tpools, err = targetPools(ctx, c, project, zone); err != nil {
		return err
	}

	return op(ctx, project, region, c, instance.SelfLink, tpools...)
}

func targetPools(ctx context.Context, c *compute.Service, project string, zone string) (_ []string, err error) {
	var (
		createdBy   string
		targetPools []string
	)

	if createdBy, err = gcloudx.InstanceGroupManagerName(); err != nil {
		return targetPools, err
	}

	if tmp, cause := igmTargetPools(ctx, c, project, zone, createdBy); cause == nil {
		err = nil
		targetPools = append(targetPools, tmp...)
	}

	if tmp, cause := rigmTargetPools(ctx, c, project, zone, createdBy); cause != nil {
		err = cause
	} else {
		err = nil
		targetPools = append(targetPools, tmp...)
	}

	return targetPools, err
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
