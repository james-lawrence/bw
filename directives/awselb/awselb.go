// Package awselb requires the following permissions in aws:
// autoscaling:DescribeAutoScalingInstances
package awselb

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/pkg/errors"
)

// LoadbalancersDetach detaches the current instance from all the loadbalancers its a part of.
func LoadbalancersDetach(ctx context.Context) (err error) {
	var (
		sess  *session.Session
		ident ec2metadata.EC2InstanceIdentityDocument
		lbs   []*elb.LoadBalancerDescription
	)

	log.Println("attempting to detach instance")

	if sess, err = session.NewSession(); err != nil {
		return errors.WithStack(err)
	}

	if ident, err = ec2metadata.New(sess).GetInstanceIdentityDocument(); err != nil {
		return errors.WithStack(err)
	}

	sess = sess.Copy(&aws.Config{
		Region: aws.String(ident.Region),
	})

	if lbs, err = loadbalancers(sess, ident); err != nil {
		return errors.WithStack(err)
	}

	elb1 := elb.New(sess)
	instances := []*elb.Instance{{InstanceId: aws.String(ident.InstanceID)}}
	for _, lb := range lbs {
		req := &elb.DeregisterInstancesFromLoadBalancerInput{LoadBalancerName: lb.LoadBalancerName, Instances: instances}
		if _, err = elb1.DeregisterInstancesFromLoadBalancerWithContext(ctx, req); err != nil {
			return errors.WithStack(err)
		}

		if err = waitForDetach(ctx, elb1, lb, ident); err != nil {
			return err
		}
	}

	log.Println("instance successfully detached")
	return nil
}

// LoadbalancersAttach attaches the current instance to all the loadbalancers its a part of.
func LoadbalancersAttach(ctx context.Context) (err error) {
	var (
		sess  *session.Session
		ident ec2metadata.EC2InstanceIdentityDocument
		lbs   []*elb.LoadBalancerDescription
	)
	log.Println("attempting to attach instance")

	if sess, err = session.NewSession(); err != nil {
		return errors.WithStack(err)
	}

	if ident, err = ec2metadata.New(sess).GetInstanceIdentityDocument(); err != nil {
		return errors.WithStack(err)
	}

	sess = sess.Copy(&aws.Config{
		Region: aws.String(ident.Region),
	})

	if lbs, err = loadbalancers(sess, ident); err != nil {
		return errors.WithStack(err)
	}

	elb1 := elb.New(sess)
	instances := []*elb.Instance{{InstanceId: aws.String(ident.InstanceID)}}
	for _, lb := range lbs {
		req := &elb.RegisterInstancesWithLoadBalancerInput{LoadBalancerName: lb.LoadBalancerName, Instances: instances}
		if _, err = elb1.RegisterInstancesWithLoadBalancerWithContext(ctx, req); err != nil {
			return errors.WithStack(err)
		}

		if err = waitForAttach(ctx, elb1, lb, ident); err != nil {
			return err
		}

		if err = waitForHealth(ctx, elb1, lb, ident); err != nil {
			return err
		}
	}

	log.Println("instance successfully attached")
	return nil
}

func loadbalancers(sess *session.Session, ident ec2metadata.EC2InstanceIdentityDocument) (lbs []*elb.LoadBalancerDescription, err error) {
	var (
		asgs     *autoscaling.AutoScaling
		dio      *elb.DescribeLoadBalancersOutput
		iao      *autoscaling.DescribeAutoScalingInstancesOutput
		instance *autoscaling.InstanceDetails
		asg      *autoscaling.DescribeAutoScalingGroupsOutput
	)

	lbs = []*elb.LoadBalancerDescription{}
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

	elb1 := elb.New(sess)

	for _, group := range asg.AutoScalingGroups {
		log.Println("group loadbalancers", aws.StringValueSlice(group.LoadBalancerNames))
		if dio, err = elb1.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{LoadBalancerNames: group.LoadBalancerNames}); err != nil {
			return lbs, errors.WithStack(err)
		}
		lbs = append(lbs, dio.LoadBalancerDescriptions...)
	}

	return lbs, err
}

type errString string

func (t errString) Error() string {
	return string(t)
}

const (
	errInstanceNotFound = errString("instance not found")
	errUnhealthy        = errString("instance is unhealthy")
)

func waitForHealth(ctx context.Context, e *elb.ELB, lbd *elb.LoadBalancerDescription, i ec2metadata.EC2InstanceIdentityDocument) (err error) {
	for {
		if err = healthyInstance(ctx, e, lbd, i); err == errUnhealthy {
			log.Println("instance unhealthy retrying")
			time.Sleep(2 * time.Second)
			continue
		}

		return errors.WithStack(err)
	}
}

func waitForAttach(ctx context.Context, elb1 *elb.ELB, lbd *elb.LoadBalancerDescription, ident ec2metadata.EC2InstanceIdentityDocument) (err error) {
	for {
		if err = hasInstance(ctx, elb1, lbd, ident); err == errInstanceNotFound {
			log.Println("instance missing retrying")
			time.Sleep(2 * time.Second)
			continue
		}

		return errors.WithStack(err)
	}
}

func waitForDetach(ctx context.Context, elb1 *elb.ELB, lbd *elb.LoadBalancerDescription, ident ec2metadata.EC2InstanceIdentityDocument) (err error) {
	for {
		if err := hasInstance(ctx, elb1, lbd, ident); err == errInstanceNotFound {
			return nil
		}
		log.Println("instance found retrying")
		time.Sleep(2 * time.Second)
	}
}

// will return nil when the instance is healthy.
func healthyInstance(ctx context.Context, e *elb.ELB, lbd *elb.LoadBalancerDescription, i ec2metadata.EC2InstanceIdentityDocument) (err error) {
	const (
		inService = "InService"
	)

	var (
		healthRequest elb.DescribeInstanceHealthInput
		health        *elb.DescribeInstanceHealthOutput
	)

	healthRequest = elb.DescribeInstanceHealthInput{
		LoadBalancerName: lbd.LoadBalancerName,
		Instances:        []*elb.Instance{{InstanceId: aws.String(i.InstanceID)}},
	}

	if health, err = e.DescribeInstanceHealthWithContext(ctx, &healthRequest); err != nil {
		return errors.WithStack(err)
	}

	for _, h := range health.InstanceStates {
		if *h.State == inService {
			return nil
		}
	}

	return errUnhealthy
}

func hasInstance(ctx context.Context, elb1 *elb.ELB, lbd *elb.LoadBalancerDescription, ident ec2metadata.EC2InstanceIdentityDocument) (err error) {
	var (
		resp *elb.DescribeLoadBalancersOutput
	)
	req := &elb.DescribeLoadBalancersInput{LoadBalancerNames: []*string{lbd.LoadBalancerName}}

	if resp, err = elb1.DescribeLoadBalancersWithContext(ctx, req); err != nil {
		return errors.WithStack(err)
	}

	for _, lbd := range resp.LoadBalancerDescriptions {
		for _, i := range lbd.Instances {
			if aws.StringValue(i.InstanceId) == ident.InstanceID {
				return nil
			}
		}
	}

	return errInstanceNotFound
}
