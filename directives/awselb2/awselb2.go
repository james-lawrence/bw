// Package awselb2 requires the following permissions in aws:
// autoscaling:DescribeAutoScalingInstances
package awselb2

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/pkg/errors"
)

// LoadbalancersDetach detaches the current instance from all the loadbalancers its a part of.
func LoadbalancersDetach() (err error) {
	var (
		sess     *session.Session
		instance *autoscaling.InstanceDetails
		lbs      []*elbv2.LoadBalancer
	)

	if sess, err = session.NewSession(); err != nil {
		return errors.WithStack(err)
	}

	if lbs, err = loadbalancers(sess); err != nil {
		return errors.WithStack(err)
	}

	elb := elbv2.New(sess)
	targets := []*elbv2.TargetDescription{{Id: instance.InstanceId}}
	for _, lb := range lbs {
		if _, err = elb.DeregisterTargets(&elbv2.DeregisterTargetsInput{TargetGroupArn: lb.LoadBalancerArn, Targets: targets}); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// LoadbalancersAttach attaches the current instance to all the loadbalancers its a part of.
func LoadbalancersAttach() (err error) {
	var (
		sess     *session.Session
		instance *autoscaling.InstanceDetails
		lbs      []*elbv2.LoadBalancer
	)

	if sess, err = session.NewSession(); err != nil {
		return errors.WithStack(err)
	}

	if lbs, err = loadbalancers(sess); err != nil {
		return errors.WithStack(err)
	}

	elb := elbv2.New(sess)

	targets := []*elbv2.TargetDescription{{Id: instance.InstanceId}}
	for _, lb := range lbs {
		if _, err = elb.RegisterTargets(&elbv2.RegisterTargetsInput{TargetGroupArn: lb.LoadBalancerArn, Targets: targets}); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func loadbalancers(sess *session.Session) (lbs []*elbv2.LoadBalancer, err error) {
	var (
		ident    ec2metadata.EC2InstanceIdentityDocument
		asgs     *autoscaling.AutoScaling
		dio      *elbv2.DescribeLoadBalancersOutput
		iao      *autoscaling.DescribeAutoScalingInstancesOutput
		instance *autoscaling.InstanceDetails
		asg      *autoscaling.DescribeAutoScalingGroupsOutput
		group    *autoscaling.Group
	)

	if ident, err = ec2metadata.New(sess, aws.NewConfig().WithRegion(ident.Region)).GetInstanceIdentityDocument(); err != nil {
		return lbs, errors.WithStack(err)
	}

	asgs = autoscaling.New(sess, aws.NewConfig().WithRegion(ident.Region))

	if iao, err = asgs.DescribeAutoScalingInstances(&autoscaling.DescribeAutoScalingInstancesInput{InstanceIds: []*string{&ident.InstanceID}}); err != nil {
		return lbs, errors.WithStack(err)
	}

	if len(iao.AutoScalingInstances) == 0 {
		return lbs, errors.Errorf("no autoscaling instance found for: %s", ident.InstanceID)
	}
	instance = iao.AutoScalingInstances[0]

	if asg, err = asgs.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: []*string{instance.AutoScalingGroupName}}); err != nil {
		return lbs, errors.WithStack(err)
	}

	if len(asg.AutoScalingGroups) == 0 {
		return lbs, errors.Errorf("no autoscaling group found for: %s", *instance.AutoScalingGroupName)
	}
	group = asg.AutoScalingGroups[0]

	elb := elbv2.New(sess)

	if dio, err = elb.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{Names: group.LoadBalancerNames, LoadBalancerArns: group.TargetGroupARNs}); err != nil {
		return lbs, errors.WithStack(err)
	}

	return dio.LoadBalancers, err
}
