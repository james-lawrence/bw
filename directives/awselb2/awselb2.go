// Package awselb2 is used to attach and detach instances from elastic load balancers.
// requires the following permissions in aws:
// autoscaling:DescribeAutoScalingInstances
// elasticloadbalancing:DescribeLoadBalancers
// elasticloadbalancing:DeregisterInstancesFromLoadBalancer
// elasticloadbalancing:RegisterInstancesWithLoadBalancer
package awselb2

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/pkg/errors"
)

// LoadbalancersDetach detaches the current instance from all the loadbalancers its a part of.
func LoadbalancersDetach(ctx context.Context) (err error) {
	var (
		sess     *session.Session
		instance *autoscaling.InstanceDetails
		lbs      []*elbv2.LoadBalancer
	)

	cfg := request.WithRetryer(aws.NewConfig(), client.DefaultRetryer{
		NumMaxRetries:    5,
		MinRetryDelay:    200 * time.Millisecond,
		MaxRetryDelay:    30 * time.Second,
		MaxThrottleDelay: 30 * time.Second,
	})

	if sess, err = session.NewSession(cfg); err != nil {
		return errors.WithStack(err)
	}

	// if we're not on an ec2 instance, nothing to do.
	if !ec2metadata.New(sess).Available() {
		return nil
	}

	if lbs, err = loadbalancers(sess); err != nil {
		return errors.WithStack(err)
	}

	elb := elbv2.New(sess)
	targets := []*elbv2.TargetDescription{{Id: instance.InstanceId}}
	for _, lb := range lbs {
		if _, err = elb.DeregisterTargetsWithContext(ctx, &elbv2.DeregisterTargetsInput{TargetGroupArn: lb.LoadBalancerArn, Targets: targets}); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// LoadbalancersAttach attaches the current instance to all the loadbalancers its a part of.
func LoadbalancersAttach(ctx context.Context) (err error) {
	var (
		sess     *session.Session
		instance *autoscaling.InstanceDetails
		lbs      []*elbv2.LoadBalancer
	)

	cfg := request.WithRetryer(aws.NewConfig(), client.DefaultRetryer{
		NumMaxRetries:    5,
		MinRetryDelay:    200 * time.Millisecond,
		MaxRetryDelay:    30 * time.Second,
		MaxThrottleDelay: 30 * time.Second,
	})

	if sess, err = session.NewSession(cfg); err != nil {
		return errors.WithStack(err)
	}

	// if we're not on an ec2 instance, nothing to do.
	if !ec2metadata.New(sess).Available() {
		return nil
	}

	if lbs, err = loadbalancers(sess); err != nil {
		return errors.WithStack(err)
	}

	elb := elbv2.New(sess)

	targets := []*elbv2.TargetDescription{{Id: instance.InstanceId}}
	for _, lb := range lbs {
		if _, err = elb.RegisterTargetsWithContext(ctx, &elbv2.RegisterTargetsInput{TargetGroupArn: lb.LoadBalancerArn, Targets: targets}); err != nil {
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
	)
	lbs = []*elbv2.LoadBalancer{}

	if ident, err = ec2metadata.New(sess, aws.NewConfig()).GetInstanceIdentityDocument(); err != nil {
		return lbs, errors.WithStack(err)
	}

	cfg := &aws.Config{
		Region: aws.String(ident.Region),
	}
	sess = sess.Copy(cfg)
	asgs = autoscaling.New(sess)

	if iao, err = asgs.DescribeAutoScalingInstances(&autoscaling.DescribeAutoScalingInstancesInput{InstanceIds: []*string{&ident.InstanceID}}); err != nil {
		return lbs, errors.WithStack(err)
	}

	if len(iao.AutoScalingInstances) == 0 {
		log.Printf("no autoscaling instance found for: %s", ident.InstanceID)
		return lbs, nil
	}
	instance = iao.AutoScalingInstances[0]

	if asg, err = asgs.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: []*string{instance.AutoScalingGroupName}}); err != nil {
		return lbs, errors.WithStack(err)
	}

	if len(asg.AutoScalingGroups) == 0 {
		log.Printf("no autoscaling group found for: %s\n, ignoring", *instance.AutoScalingGroupName)
		return lbs, nil
	}

	elb := elbv2.New(sess)

	for _, group := range asg.AutoScalingGroups {
		log.Println("group loadbalancers", aws.StringValueSlice(group.LoadBalancerNames))
		log.Println("group target group arms", aws.StringValueSlice(group.TargetGroupARNs))
		if dio, err = elb.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{Names: group.LoadBalancerNames}); err != nil {
			return lbs, errors.WithStack(err)
		}

		lbs = append(lbs, dio.LoadBalancers...)
	}

	return lbs, err
}
