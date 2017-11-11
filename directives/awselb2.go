package directives

import (
	"io"

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

// Ext extensions to succeed against.
func (AWSELB2AttachLoader) Ext() []string {
	return []string{".attach-awselb2"}
}

// Build builds a directive from the reader.
func (t AWSELB2AttachLoader) Build(r io.Reader) (Directive, error) {
	return closure(awselb2.LoadbalancersAttach), nil
}

// AWSELB2DetachLoader directive.
type AWSELB2DetachLoader struct{}

// Ext extensions to succeed against.
func (AWSELB2DetachLoader) Ext() []string {
	return []string{".detach-awselb2"}
}

// Build builds a directive from the reader.
func (t AWSELB2DetachLoader) Build(r io.Reader) (Directive, error) {
	return closure(awselb2.LoadbalancersDetach), nil
}
