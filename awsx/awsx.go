package awsx

import (
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/pkg/errors"
)

// AutoscalingPeers return a list of peers for this instance based on the autoscaling group.
// errors out if no autoscaling group is associated with the instance.
func AutoscalingPeers() (peers []ec2.Instance, err error) {
	var (
		sess     *session.Session
		ident    ec2metadata.EC2InstanceIdentityDocument
		asgs     *autoscaling.AutoScaling
		iao      *autoscaling.DescribeAutoScalingInstancesOutput
		instance *autoscaling.InstanceDetails
		asg      *autoscaling.DescribeAutoScalingGroupsOutput
		group    *autoscaling.Group
		ec2io    *ec2.DescribeInstancesOutput
		peersID  []*string
	)

	if sess, err = session.NewSession(); err != nil {
		return peers, errors.WithStack(err)
	}

	if ident, err = ec2metadata.New(sess).GetInstanceIdentityDocument(); err != nil {
		return peers, errors.WithStack(err)
	}
	asgs = autoscaling.New(sess)

	if iao, err = asgs.DescribeAutoScalingInstances(&autoscaling.DescribeAutoScalingInstancesInput{InstanceIds: []*string{&ident.InstanceID}}); err != nil {
		return peers, errors.WithStack(err)
	}

	if len(iao.AutoScalingInstances) == 0 {
		return peers, errors.Errorf("no autoscaling instance found for: %s", ident.InstanceID)
	}
	instance = iao.AutoScalingInstances[0]

	if asg, err = asgs.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: []*string{instance.AutoScalingGroupName}}); err != nil {
		return peers, errors.WithStack(err)
	}

	if len(asg.AutoScalingGroups) == 0 {
		return peers, errors.Errorf("no autoscaling group found for: %s", *instance.AutoScalingGroupName)
	}
	group = asg.AutoScalingGroups[0]

	peersID = make([]*string, 0, len(group.Instances))
	for _, i := range group.Instances {
		peersID = append(peersID, i.InstanceId)
	}

	if ec2io, err = ec2.New(sess).DescribeInstances(&ec2.DescribeInstancesInput{InstanceIds: peersID}); err != nil {
		return peers, errors.WithStack(err)
	}

	for _, r := range ec2io.Reservations {
		for _, i := range r.Instances {
			peers = append(peers, *i)
		}
	}

	return peers, nil
}

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

	if ident, err = ec2metadata.New(sess).GetInstanceIdentityDocument(); err != nil {
		return lbs, errors.WithStack(err)
	}

	asgs = autoscaling.New(sess)
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
