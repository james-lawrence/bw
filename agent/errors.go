package agent

import "github.com/james-lawrence/bw/internal/x/errorsx"

// ErrDisabledMachine returned when the state machine interface is disabled.
const ErrDisabledMachine = errorsx.String("this node is not a member of the quorum")
