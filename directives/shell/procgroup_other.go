//go:build !unix

package shell

import "os/exec"

func stopWithChildren(cmd *exec.Cmd) {}
