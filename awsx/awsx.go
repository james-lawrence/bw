package awsx

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
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

	sess = sess.Copy(&aws.Config{
		Region: aws.String(ident.Region),
	})

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
