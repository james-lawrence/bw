package gcloudx

import (
	"strings"

	"cloud.google.com/go/compute/metadata"
	"github.com/pkg/errors"
	"google.golang.org/api/compute/v1"
)

func InstanceGroupManagerName() (createdBy string, err error) {
	if createdBy, err = metadata.InstanceAttributeValue("created-by"); err != nil {
		return createdBy, errors.WithStack(err)
	}

	if r := PathSuffix(createdBy, "/"); r == "" {
		return "", errors.New("unable to extract instance group manager name from path")
	} else {
		createdBy = r
	}

	return createdBy, nil
}

func PathSuffix(s string, sep string) string {
	var (
		idx int
	)

	if idx = strings.LastIndex(s, sep); idx < 0 || idx > len(s)-1 {
		return ""
	}

	return s[idx+1:]
}

func DistributionPolicyZones(policy *compute.DistributionPolicy) (res []string) {
	if policy == nil {
		return res
	}

	for _, z := range policy.Zones {
		res = append(res, PathSuffix(z.Zone, "/"))
	}

	return res
}

func ZonalRegion(s string, sep string) string {
	var (
		idx int
	)

	if idx = strings.LastIndex(s, sep); idx < 0 || idx > len(s)-1 {
		return ""
	}

	return s[:idx]
}
