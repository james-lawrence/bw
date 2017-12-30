package directives

import (
	"io"

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

// Ext extensions to succeed against.
func (AWSELBAttachLoader) Ext() []string {
	return []string{".attach-awselb"}
}

// Build builds a directive from the reader.
func (t AWSELBAttachLoader) Build(r io.Reader) (Directive, error) {
	return closure(awselb.LoadbalancersAttach), nil
}

// AWSELBDetachLoader directive.
type AWSELBDetachLoader struct{}

// Ext extensions to succeed against.
func (AWSELBDetachLoader) Ext() []string {
	return []string{".detach-awselb"}
}

// Build builds a directive from the reader.
func (t AWSELBDetachLoader) Build(r io.Reader) (Directive, error) {
	return closure(awselb.LoadbalancersDetach), nil
}
