### Bearded Wookie configuration management.

secure, fast, reliable, and self contained configuration management.

#### security

bearded-wookie uses TLS to encrypt all data transferred between agents and clients.

#### project goals - how features are decided on and accepted. (no particular order)

- be a secure, resource efficient, and reliable configuration management tool.
- no centralized server.
- no required infrastructure. any additional infrastructure should be optional.
- low operational complexity. a.k.a easy to use and operate.

### quick start - debian package installation

see examples for configuration.

```
sudo add-apt-repository ppa:jljatone/bw
sudo apt-get update
sudo apt-get install bearded-wookie
```

### quick start - local development (assumes linux).

```bash
# install bearded wookie
go install github.com/james-lawrence/bw/cmd/...

# initialize user credentials.
bw me init

# use bearded wookie to bootstrap the local development cluster
rm -rf ~/.cache/bearded-wookie/agent* && bw deploy local linux-dev

systemctl --user restart bearded-wookie-deploy-notifications@agent1.service
systemctl --user restart bearded-wookie@agent{1,2,3,4}.service
bw deploy env linux-test --insecure
```

### quick start - gcloud (assumes a gcloud project, and terraform).

IMPORTANT: quickstart examples don't configure a VPN, a VPN is highly recommended for production environments.

```bash
# install bearded wookie
go install github.com/james-lawrence/bw/cmd/...

# initialize user credentials.
bw me init

# setup default gcloud project
gcloud auth application-default login

pushd .examples/gcloud && terraform init && popd
pushd .examples/gcloud && terraform destroy && terraform apply && popd
pushd .examples && bw deploy example && popd
```

example commands:

- `bw workspace bootstrap` creates a deployment workspace. this is a directory + skeleton.
- `bw environment create {name} {address}` creates a new environment within the workspace.
- `bw deploy {environment}` deploy to the specified environment.
- `bw deploy --canary {environment}` deploy to a single consistent server.
- `bw deploy --ip='127.0.0.1' {environment}` deploy to the servers that match the given filters.
- `bw deploy archive {environment} {deploymentID}` redeploy a previously uploaded archive.
- `bw deploy archive --ip='127.0.0.1' {environment} {deploymentID}` filter a redeploy to specific servers.
- `bw info check {address}:{port}` checks if the cluster is reachable.

### architecture overview

bearded-wookie is built off 4 main protocols.

- SWIM - a gossip protocol for discovering peers and agent health checks.
- RAFT - consensus algorithm for shared state, e.g.) current deployment.
- ACME protocol - for bootstrapping TLS.
- bittorrent - for transferring deployment archives between servers.

by using these 4 protocols bearded-wookie avoids needing a centralized server, while remaining durable and easy to operate.

Benefits of bearded-wookie:

- can work entirely inside of a VPN; no need to expose anything outside of the network. (requires using ACME DNS challenge, or a custom certificate authority)
- durable. unless you literally lose all of the nodes and a majority of your entire cluster simultaneously the cluster will continue to operate.
  - bw does have support for remote bootstrapping, its just not required for normal operation.
- builtin support for custom bootstrapping services. this allows you to lose a majority of your servers and continue to operating.
  - see bootstrap/filesystem.go for an example. the filesystem example allows you to mount NFS to store the last successful deployment.
- Lower costs by not needing additional infrastructure just to support deployments.
- when something does go wrong, bw is easy to repair. just destroy its cache (default /var/cache/bearded-wookie) and reboot the agents.
- simple configuration simplest configuration is ~ 40 lines between the agent/clients, one port (tcp and udp), and DNS record.

## Single-Node Deployment Architecture & Troubleshooting

This section provides a deep dive into configuring BW for single-node deployments, common pitfalls, and debugging techniques based on real-world implementation experience.

### Architecture Overview: Single-Node vs Multi-Node

BW is designed for distributed multi-node clusters, but can be configured for single-node operation with specific settings:

```
Multi-Node (Production)          Single-Node (Development/Testing)
┌─────────────────────────┐      ┌─────────────────────────┐
│ Client                  │      │ Client                  │
│ ├── bw deploy env       │      │ ├── bw deploy env       │
│ └── TLS verification    │      │ └── --insecure flag     │
└─────────────────────────┘      └─────────────────────────┘
            │                                   │
            ▼                                   ▼
┌─────────────────────────┐      ┌─────────────────────────┐
│ Load Balancer/Proxy     │      │ Optional Nginx Proxy    │
│ ├── Port 443 (HTTPS)   │      │ ├── Port 443 → 2000     │
│ └── ALPN Routing        │      │ └── Stream mode         │
└─────────────────────────┘      └─────────────────────────┘
            │                                   │
    ┌───────┼───────┐                          ▼
    ▼       ▼       ▼              ┌─────────────────────────┐
┌───────┐ ┌───────┐ ┌───────┐      │ Single BW Agent         │
│Agent 1│ │Agent 2│ │Agent N│      │ ├── Port 2000           │
│:2000  │ │:2000  │ │:2000  │      │ ├── SWIM: self-gossip  │
└───────┘ └───────┘ └───────┘      │ ├── RAFT: single voter  │
    │       │       │              │ ├── P2P: 127.0.0.1     │
    └───────┼───────┘              │ └── minimumNodes: 1     │
            │                      └─────────────────────────┘
┌─────────────────────────┐
│ Distributed P2P Mesh    │
│ ├── Cross-node gossip   │
│ ├── RAFT consensus      │
│ └── BitTorrent sharing  │
└─────────────────────────┘
```

### Critical Configuration for Single-Node Deployments

#### 1. Agent Configuration (`agent.config`)

```yaml
root: "/var/cache/bearded-wookie"
keepN: 5 # Deployment history retention
minimumNodes: 1 # ⚠️  CRITICAL: disable quorum consensus.
snapshotFrequency: "3h" # RAFT snapshot interval
credentials:
  source: disabled # ⚠️  SECURITY: this is not recommended. but can be useful for debugging purposes.
notary:
  authority: [
      "/etc/bearded-wookie/default/bw.auth.keys", # SSH key authentication
    ]
clusterTokens: # Unique cluster identification
  - "cluster-token-1"
  - "cluster-token-2"
```

#### 2. Network Environment (`agent.env`)

```bash
# P2P Discovery and Communication
BEARDED_WOOKIE_AGENT_CLUSTER_P2P_DISCOVERY_PORT=2000

# Network Binding Configuration
BEARDED_WOOKIE_AGENT_P2P_BIND=0.0.0.0:2000           # Listen on all interfaces
BEARDED_WOOKIE_AGENT_P2P_ADVERTISED=127.0.0.1:2000   # ⚠️  NOTE: this depends on your cloud infrastructure.

# Cluster Identity
# BEARDED_WOOKIE_AGENT_SERVERNAME=your-node-name      # you should almost never need to do this. but can be handy for debugging.

# Cloud Integration
BEARDED_WOOKIE_AGENT_CLUSTER_PEERS_GCLOUD_POOL=false # Disable auto-discovery of peers

# Execution Environment
SHELL=/bin/bash
```

#### 3. Client Configuration (`.bwconfig/environment/config.yml`)

```yaml
address: "your-node:2000" # External agent address
discovery: "your-node:2000" # Same as address for single-node
servername: "your-node-name" # Must match agent
concurrency: 0.25 # Deployment parallelism
credentials:
  source: disabled # ⚠️  CRITICAL: Match agent setting
deploy:
  dir: ".deploy/environment" # Deployment scripts location
  timeout: 1h0m0s # Deployment timeout
  treeish: origin/main # Git reference to deploy
```

### Common Errors and Debugging Techniques

#### Error 1: "insufficient nodes for deployment"

**Symptoms:**

```
[ERROR] deployment rejected: cluster has 1 nodes, minimum required: 3
```

**Root Cause:** BW's default safety mechanism requires multiple nodes for consensus.

**Solution:**

```yaml
# In agent.config
minimumNodes: 1 # Override default minimum
```

**Debugging:**

```bash
# Check cluster status
bw info nodes environment-name --insecure -v

# Verify agent configuration
grep -i minimum /etc/bearded-wookie/default/agent.config
```

#### Error 2: "tls: failed to verify certificate"

**Symptoms:**

```
[ERROR] x509: certificate signed by unknown authority
[ERROR] failed to verify certificate: x509: certificate is not valid
```

**Root Cause:** Self-signed certificates don't match between client and server.

**Solution:**

```bash
# Agent side: disable TLS verification
credentials:
  source: disabled

# Client side: use insecure flag
bw deploy env environment-name --insecure
```

**Debugging:**

```bash
# Test raw connectivity
openssl s_client -connect your-node:2000 -verify_return_error

# Check certificate details
openssl x509 -in cert.pem -text -noout

# Verify BW can connect
bw info check your-node:2000 --insecure -v
```

#### Error 3: Network binding and discovery issues

**Symptoms:**

```
[ERROR] failed to bind to address 127.0.0.1:2000: address already in use
[ERROR] no peers discovered after timeout
```

**Root Cause:** Incorrect bind vs advertised address configuration.

**Solution:**

```bash
# Bind to all interfaces for external access
BEARDED_WOOKIE_AGENT_P2P_BIND=0.0.0.0:2000

# Advertise localhost for single-node internal communication
BEARDED_WOOKIE_AGENT_P2P_ADVERTISED=127.0.0.1:2000

# Ensure discovery port matches bind port
BEARDED_WOOKIE_AGENT_CLUSTER_P2P_DISCOVERY_PORT=2000
```

**Debugging:**

```bash
# Check port bindings
netstat -tlnp | grep :2000
ss -tlnp | grep :2000

# Test connectivity from client
telnet your-node 2000
nc -zv your-node 2000

# Check agent logs
journalctl -u bearded-wookie-agent -f
```

#### Error 5: Go build environment issues

**Symptoms:**

```
[LOCAL] executing go build ...
[LOCAL] go: module cache not found: neither GOMODCACHE nor GOPATH is set
[LOCAL] failed directive: build.bwcmd
```

**Root Cause:** Missing Go environment variables during build.

**Solution:**

```bash
# In build script (.local/build.bwcmd)
- command: cd %bw.temp.directory%/archive && go build -o %bw.archive.directory%/.filesystem/usr/local/bin/daemon ./cmd/daemon
  loadenv:
    - "%bw.archive.directory%/bw.env"

# In bw.env file
CGO_ENABLED=0
GOOS=linux
GOARCH=amd64
GOPATH=/tmp/go
GOCACHE=/tmp/go-cache
```

**Debugging:**

```bash
# Check Go environment during build
- command: env | grep GO | sort

# Test build manually
cd archive && CGO_ENABLED=0 GOOS=linux go build ./cmd/daemon
```

### Deployment Directory Structure

Proper deployment structure following established patterns:

```
.deploy/environment/
├── .local/                    # Local build scripts
│   ├── 01-archive.bwcmd      # Git archive creation
│   └── 02-build.bwcmd        # Binary compilation
├── .remote/                   # Remote deployment scripts
│   ├── 00-unpack.bwcmd       # Archive extraction
│   ├── 01-services.bwcmd     # Service configuration
│   └── 90-systemd.bwcmd      # Service management
├── .filesystem/               # Files for deployment
│   └── usr/local/bin/         # Compiled binaries
├── systemd/                   # Systemd service files
│   └── service.service
├── bw.env                     # Build environment
└── .gitignore                 # Exclude build artifacts
```

### Nginx Integration Patterns

#### Simple TCP Proxy

```nginx
stream {
    server {
        listen 443;
        proxy_pass 127.0.0.1:2000;
        proxy_timeout 1s;
        proxy_responses 1;
    }
}
```

#### ALPN Protocol Routing

```nginx
stream {
    map $ssl_preread_alpn_protocols $upstream {
        ~\bacme-tls/1\b 127.0.0.1:2000; # BW tls management
		      ~\bbw.mux\b 127.0.0.1:2000;     # BW protocol traffic
		      ~\bbw.proxy\b 127.0.0.1:2000;   # BW proxy traffic
        default  127.0.0.1:8080;        # Application traffic
    }

    server {
        listen 443;
        ssl_preread on;
        proxy_pass $upstream;
    }
}
```

### Production Migration Path

To migrate from single-node to multi-node:

1. **Add additional nodes** with same cluster tokens
2. **Update `minimumNodes`** to match cluster size
3. **Enable proper TLS** with `credentials.source: acme` or custom CA
4. **Configure load balancing** across all agents
5. **Update client config** to use multiple discovery addresses

### Monitoring and Observability

Key metrics to monitor for single-node deployments:

```bash
# Agent health
curl -s http://localhost:2000/healthz

# Deployment history
bw info deployments environment-name --insecure

# Cluster status
bw info nodes environment-name --insecure

# Service status
systemctl status bearded-wookie-agent
journalctl -u bearded-wookie-agent --since="1 hour ago"
```
