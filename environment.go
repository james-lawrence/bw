package bw

// defines available environment variables for configuration
const (
	EnvLogsVerbose        = "BEARDED_WOOKIE_LOGS_VERBOSE"         // enable verbose logging. boolean, see strconv.ParseBool for valid values.
	EnvLogsGossip         = "BEARDED_WOOKIE_LOGS_GOSSIP"          // enable logging for gossip protocol. boolean, see strconv.ParseBool for valid values.
	EnvLogsRaft           = "BEARDED_WOOKIE_LOGS_RAFT"            // enable logging for the raft protocol. boolean, see strconv.ParseBool for valid values.
	EnvLogsGRPC           = "BEARDED_WOOKIE_LOGS_GRPC"            // enable logging for grpc protocol. boolean, see strconv.ParseBool for valid values.
	EnvLogsConfiguration  = "BEARDED_WOOKIE_LOGS_CONFIGURATION"   // enable logging for configuration. boolean, see strconv.ParseBool for valid values.
	EnvDisplayName        = "BEARDED_WOOKIE_DISPLAY_NAME"         // environment variable to determine display name to be used, defaults to current user's name.
	EnvAgentDiscoveryBind = "BEARDED_WOOKIE_AGENT_DISCOVERY_BIND" // environment variable to specify the network address to listen to. e.g.) 0.0.0.0:2001
	EnvAgentRPCBind       = "BEARDED_WOOKIE_AGENT_RPC_BIND"       // environment variable to specify the network address to listen to. e.g.) 0.0.0.0:2002
	EnvAgentSWIMBind      = "BEARDED_WOOKIE_AGENT_CLUSTER_BIND"   // environment variable to specify the network address to listen to. e.g.) 0.0.0.0:2003
	EnvAgentRAFTBind      = "BEARDED_WOOKIE_AGENT_RAFT_BIND"      // environment variable to specify the network address to listen to. e.g.) 0.0.0.0:2004
	EnvAgentTorrentBind   = "BEARDED_WOOKIE_AGENT_TORRENT_BIND"   // environment variable to specify the network address to listen to. e.g.) 0.0.0.0:2005
	EnvAgentAutocertBind  = "BEARDED_WOOKIE_AGENT_AUTOCERT_BIND"  // environment variable to specify the network address to listen to. e.g.) 0.0.0.0:2006
)
