// Package awselb requires the following permissions in aws:
// autoscaling:DescribeAutoScalingInstances
package awselb

import (
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
func LoadbalancersDetach() (err error) {
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
		if _, err = elb1.DeregisterInstancesFromLoadBalancer(req); err != nil {
			return errors.WithStack(err)
		}

		if err = waitForDetach(elb1, lb, ident); err != nil {
			return err
		}
	}

	log.Println("instance successfully detached")
	return nil
}

// LoadbalancersAttach attaches the current instance to all the loadbalancers its a part of.
func LoadbalancersAttach() (err error) {
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
		if _, err = elb1.RegisterInstancesWithLoadBalancer(req); err != nil {
			return errors.WithStack(err)
		}

		if err = waitForAttach(elb1, lb, ident); err != nil {
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

const errInstanceNotFound = errString("instance not found")

func waitForAttach(elb1 *elb.ELB, lbd *elb.LoadBalancerDescription, ident ec2metadata.EC2InstanceIdentityDocument) (err error) {
	for {
		if err = hasInstance(elb1, lbd, ident); err == errInstanceNotFound {
			log.Println("instance missing retrying")
			time.Sleep(2 * time.Second)
			continue
		}

		return errors.WithStack(err)
	}
}

func waitForDetach(elb1 *elb.ELB, lbd *elb.LoadBalancerDescription, ident ec2metadata.EC2InstanceIdentityDocument) (err error) {
	for {
		if err := hasInstance(elb1, lbd, ident); err == errInstanceNotFound {
			return nil
		}
		log.Println("instance found retrying")
		time.Sleep(2 * time.Second)
	}
}

func hasInstance(elb1 *elb.ELB, lbd *elb.LoadBalancerDescription, ident ec2metadata.EC2InstanceIdentityDocument) (err error) {
	var (
		resp *elb.DescribeLoadBalancersOutput
	)
	req := &elb.DescribeLoadBalancersInput{LoadBalancerNames: []*string{lbd.LoadBalancerName}}

	if resp, err = elb1.DescribeLoadBalancers(req); err != nil {
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
