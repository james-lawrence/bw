package bw

// defines available environment variables for configuration
const (
	EnvLogsVerbose = "BEARDED_WOOKIE_LOGS_VERBOSE" // enable verbose logging. boolean, see strconv.ParseBool for valid values.
	EnvLogsGossip  = "BEARDED_WOOKIE_LOGS_GOSSIP"  // enable logging for gossip protocol. boolean, see strconv.ParseBool for valid values.
	EnvLogsRaft    = "BEARDED_WOOKIE_LOGS_RAFT"    // enable logging for the raft protocol. boolean, see strconv.ParseBool for valid values.
)
