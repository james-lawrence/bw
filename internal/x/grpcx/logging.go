package grpcx

import (
	"io/ioutil"
	"os"
	"strconv"

	"google.golang.org/grpc/grpclog"
)

func NewLogger() grpclog.LoggerV2 {
	errorW := ioutil.Discard
	warningW := ioutil.Discard
	infoW := ioutil.Discard

	logLevel := os.Getenv("GRPC_GO_LOG_SEVERITY_LEVEL")
	switch logLevel {
	case "", "ERROR", "error": // If env is unset, set level to ERROR.
		errorW = os.Stderr
	case "WARNING", "warning":
		warningW = os.Stderr
	case "INFO", "info":
		infoW = os.Stderr
	}

	var v int
	vLevel := os.Getenv("GRPC_GO_LOG_VERBOSITY_LEVEL")
	if vl, err := strconv.Atoi(vLevel); err == nil {
		v = vl
	}
	return grpclog.NewLoggerV2WithVerbosity(infoW, warningW, errorW, v)
}
