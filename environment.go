package bw

// defines available environment variables for configuration
const (
	EnvLogsDeploy                        = "BEARDED_WOOKIE_LOGS_DEPLOY"                                // enable verbose logging. boolean, see strconv.ParseBool for valid values.
	EnvLogsVerbose                       = "BEARDED_WOOKIE_LOGS_VERBOSE"                               // enable verbose logging. boolean, see strconv.ParseBool for valid values.
	EnvLogsGossip                        = "BEARDED_WOOKIE_LOGS_GOSSIP"                                // enable logging for gossip protocol. boolean, see strconv.ParseBool for valid values.
	EnvLogsRaft                          = "BEARDED_WOOKIE_LOGS_RAFT"                                  // enable logging for the raft protocol. boolean, see strconv.ParseBool for valid values.
	EnvLogsGRPC                          = "BEARDED_WOOKIE_LOGS_GRPC"                                  // enable logging for grpc protocol. boolean, see strconv.ParseBool for valid values.
	EnvLogsQuorum                        = "BEARDED_WOOKIE_LOGS_Quorum"                                // enable logging around the quorum state machine. boolean, see strconv.ParseBool for valid values.
	EnvLogsTLS                           = "BEARDED_WOOKIE_LOGS_TLS"                                   // enable logging for tls credentials. boolean, see strconv.ParseBool for valid values.
	EnvLogsConfiguration                 = "BEARDED_WOOKIE_LOGS_CONFIGURATION"                         // enable logging for configuration. boolean, see strconv.ParseBool for valid values.
	EnvDisplayName                       = "BEARDED_WOOKIE_DISPLAY_NAME"                               // environment variable to determine display name to be used, defaults to current user's name.
	EnvAgentP2PAdvertised                = "BEARDED_WOOKIE_AGENT_P2P_ADVERTISED"                       // environment variable to specify the network address to advertise to peers. e.g.) 127.0.0.1:2000
	EnvAgentP2PBind                      = "BEARDED_WOOKIE_AGENT_P2P_BIND"                             // environment variable to specify the network address to listen to. e.g.) 0.0.0.0:2000
	EnvAgentP2PAlternatesBind            = "BEARDED_WOOKIE_AGENT_P2P_ALTERNATES"                       // environment variable to specify the network address to listen to. e.g.) 127.0.0.1:2000
	EnvAgentClusterBootstrap             = "BEARDED_WOOKIE_AGENT_BOOTSTRAP"                            // environment variable to specify the tcp address to connect to allowing for bootstrapping.
	EnvAgentClusterPassiveCheckin        = "BEARDED_WOOKIE_AGENT_CLUSTER_PASSIVE_CHECKIN"              // environment variable to adjust the passive checking rate for the leader node.
	EnvAgentClusterEnableAWSAutoscaling  = "BEARDED_WOOKIE_AGENT_CLUSTER_PEERS_AWS_AUTOSCALING_GROUPS" // enable aws autoscale group peer detection
	EnvAgentClusterEnableGoogleCloudPool = "BEARDED_WOOKIE_AGENT_CLUSTER_PEERS_GCLOUD_POOL"            // enable gcloud pool peer detection
	EnvAgentClusterEnableDNS             = "BEARDED_WOOKIE_AGENT_CLUSTER_PEERS_DNS"                    // enable dns peer detection
	EnvAgentClusterP2PDiscoveryPort      = "BEARDED_WOOKIE_AGENT_CLUSTER_P2P_DISCOVERY_PORT"           // override the p2p discovery port
)
