package directives

import (
	"github.com/james-lawrence/bw/directives/awselb"
)

// NewAWSELBAttach ...
func NewAWSELBAttach() AWSELBAttachLoader {
	return AWSELBAttachLoader{}
}

// NewAWSELBDetach ...
func NewAWSELBDetach() AWSELBDetachLoader {
	return AWSELBDetachLoader{}
}

// AWSELBAttachLoader directive.
type AWSELBAttachLoader struct{}

// Load elb attach directive
func (t AWSELBAttachLoader) Load(path string) (dir Directive, err error) {
	if err = LoadsExtensions(path, "attach-awselb"); err != nil {
		return dir, err
	}

	return closure(awselb.LoadbalancersAttach), nil
}

// AWSELBDetachLoader directive.
type AWSELBDetachLoader struct{}

// Load elb detach directive
func (t AWSELBDetachLoader) Load(path string) (dir Directive, err error) {
	if err = LoadsExtensions(path, "detach-awselb"); err != nil {
		return dir, err
	}

	return closure(awselb.LoadbalancersDetach), nil
}
