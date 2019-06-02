package directives

import (
	"github.com/james-lawrence/bw/directives/awselb2"
)

// NewAWSELB2Attach ...
func NewAWSELB2Attach() AWSELB2AttachLoader {
	return AWSELB2AttachLoader{}
}

// NewAWSELB2Detach ...
func NewAWSELB2Detach() AWSELB2DetachLoader {
	return AWSELB2DetachLoader{}
}

// AWSELB2AttachLoader directive.
type AWSELB2AttachLoader struct{}

// Load elb detach directive
func (t AWSELB2AttachLoader) Load(path string) (dir Directive, err error) {
	if err = LoadsExtensions(path, "attach-awselb2"); err != nil {
		return dir, err
	}

	return closure(awselb2.LoadbalancersAttach), nil
}

// AWSELB2DetachLoader directive.
type AWSELB2DetachLoader struct{}

// Load elb detach directive
func (t AWSELB2DetachLoader) Load(path string) (dir Directive, err error) {
	if err = LoadsExtensions(path, "detach-awselb2"); err != nil {
		return dir, err
	}

	return closure(awselb2.LoadbalancersDetach), nil
}
