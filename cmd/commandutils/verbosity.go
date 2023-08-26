package commandutils

import (
	"fmt"
	"log"
	"os"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/grpcx"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"google.golang.org/grpc/grpclog"
)

// LogCause returns a string format based on the verbosity.
func LogCause(err error) error {
	type NotificationError interface {
		Notification()
	}

	type ShortError interface {
		UserFriendly()
	}

	var (
		nErr NotificationError
		sErr ShortError
	)

	if err == nil {
		return nil
	}

	if errors.As(err, &nErr) {
		log.Println(err)
	} else if errors.As(err, &sErr) {
		log.Println(aurora.NewAurora(true).Red("ERROR"), err)
	} else {
		log.Println("DERP")
		_ = log.Output(2, fmt.Sprintf("%T - [%+v]\n", err, err))
	}

	return err
}

func LogEnv(verbosity int) {
	switch verbosity {
	case 4:
		os.Setenv(bw.EnvLogsVerbose, "1")
	case 3:
		os.Setenv(bw.EnvLogsGRPC, "1")
		os.Setenv(bw.EnvLogsGossip, "1")
		os.Setenv(bw.EnvLogsRaft, "1")
		fallthrough
	case 2:
		os.Setenv(bw.EnvLogsVerbose, "1")
		fallthrough
	case 1:
		os.Setenv(bw.EnvLogsConfiguration, "1")
	default:
	}

	// enable GRPC logging
	if envx.Boolean(false, bw.EnvLogsGRPC, bw.EnvLogsVerbose) {
		os.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "99")
		os.Setenv("GRPC_GO_LOG_SEVERITY_LEVEL", "info")
		grpclog.SetLoggerV2(grpcx.NewLogger())
	}
}
