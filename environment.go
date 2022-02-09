package bw

// defines available environment variables for configuration
const (
	EnvLogsVerbose                       = "BEARDED_WOOKIE_LOGS_VERBOSE"                     // enable verbose logging. boolean, see strconv.ParseBool for valid values.
	EnvLogsGossip                        = "BEARDED_WOOKIE_LOGS_GOSSIP"                      // enable logging for gossip protocol. boolean, see strconv.ParseBool for valid values.
	EnvLogsRaft                          = "BEARDED_WOOKIE_LOGS_RAFT"                        // enable logging for the raft protocol. boolean, see strconv.ParseBool for valid values.
	EnvLogsGRPC                          = "BEARDED_WOOKIE_LOGS_GRPC"                        // enable logging for grpc protocol. boolean, see strconv.ParseBool for valid values.
	EnvLogsConfiguration                 = "BEARDED_WOOKIE_LOGS_CONFIGURATION"               // enable logging for configuration. boolean, see strconv.ParseBool for valid values.
	EnvDisplayName                       = "BEARDED_WOOKIE_DISPLAY_NAME"                     // environment variable to determine display name to be used, defaults to current user's name.
	EnvAgentP2PAdvertised                = "BEARDED_WOOKIE_AGENT_P2P_ADVERTISED"             // environment variable to specify the network ip to advertise to peers. e.g.) 127.0.0.1:2000
	EnvAgentP2PBind                      = "BEARDED_WOOKIE_AGENT_P2P_BIND"                   // environment variable to specify the network address to listen to. e.g.) 0.0.0.0:2000
	EnvAgentP2PAlternatesBind            = "BEARDED_WOOKIE_AGENT_P2P_ALTERNATES"             // environment variable to specify the network address to listen to. e.g.) 0.0.0.0:2000
	EnvAgentDiscoveryBind                = "BEARDED_WOOKIE_AGENT_DISCOVERY_BIND"             // environment variable to specify the network address to listen to. e.g.) 0.0.0.0:2002
	EnvAgentSWIMBind                     = "BEARDED_WOOKIE_AGENT_CLUSTER_BIND"               // environment variable to specify the network address to listen to. e.g.) 0.0.0.0:2004
	EnvAgentRAFTBind                     = "BEARDED_WOOKIE_AGENT_RAFT_BIND"                  // environment variable to specify the network address to listen to. e.g.) 0.0.0.0:2005
	EnvAgentClusterBootstrap             = "BEARDED_WOOKIE_AGENT_BOOTSTRAP"                  // environment variable to specify the tcp address to connect to allowing for bootstrapping.
	EnvAgentClusterPassiveCheckin        = "BEARDED_WOOKIE_AGENT_CLUSTER_PASSIVE_CHECKIN"    // environment variable to adjust the passive checking rate for the leader node.
	EnvAgentClusterEnableGoogleCloudPool = "BEARDED_WOOKIE_AGENT_CLUSTER_PEERS_GCLOUD_POOL"  // enable gcloud pool peer detection
	EnvAgentClusterEnableDNS             = "BEARDED_WOOKIE_AGENT_CLUSTER_PEERS_DNS"          // enable dns peer detection
	EnvAgentClusterP2PDiscoveryPort      = "BEARDED_WOOKIE_AGENT_CLUSTER_P2P_DISCOVERY_PORT" // override the p2p discovery port
)
