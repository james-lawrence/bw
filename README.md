### Bearded Wookie configuration management.
secure, fast, reliable, and self contained configuration management.

#### security
bearded-wookie uses SSL/TLS to encrypt all data transferred between agents and clients.
with one caveat - archive transfer between agents is done via torrents, they'll eventually support TLS.

#### project goals - how features are decided on and accepted. (no particular order)
- be a secure, resource efficient, and reliable configuration management tool.
- no centralized server.
- no required infrastructure. any additional infrastructure should be optional.
- support different deployment strategies. %age, batch, one at a time.
- ease of use. mainly around deployment and initial setup.

### quick start - assumes you have a gcloud project. (aws works as well)
IMPORTANT: quickstart example doesn't include a VPN, a VPN is highly recommended.
```bash
pushd .examples/gcloud && terraform init && terraform apply && bw deploy example
```

example commands:  
 - `bw environment create {name} {address}`  
 - `bw workspace bootstrap` creates a deployment workspace. this is a directory + skeleton.  
 - `bw agent` runs a bearded-wookie agent.  
 - `bw deploy {environment}` deploy to the production environment  
 - `bw deploy filtered --ip='10.142.0.1' {environment}` to the servers that match the given filters.  
 - `bw info {environment}` display information and receive events about the environment.  
 - `bw info check {address}:2001` checks if the cluster is reachable.

### architecture overview
bearded-wookie is built off 4 main protocols.
- SWIM - a gossip protocol for discovering peers and agent health checks.
- RAFT - consensus algorithm for shared state, such as current deployment and user credentials.
- Bit torrent - for archive transfer between nodes.
- ACME protocol - for bootstrapping TLS.

by using these 4 protocols bearded-wookie avoids needing a centralized server, while remaining durable.
it also makes bearded-wookie easy to operate.

Benefits of bearded-wookie:
- No need for remote storage, your servers already have a copy of the latest deploy, no need for a backup unless you literally lose all of the leader nodes and a majority of your entire cluster simultaneously - and honestly if that happens you have bigger issues. (note: BW does support a remote backup for bootstrapping, its just not required for normal operation)
- Lower costs by not needing additional infrastructure just to support deployments.
- when something does go wrong, BW is easy to repair. just destroy its cache (default /var/cache/bearded-wookie/deploys) and reboot the agents.
- support single box deployments - canary deploys can be incredibly useful.
- builtin support for custom bootstrapping - see bootstrap/filesystem.go for an example. the filesystem archives the latest successful deploy to a directory and returns that result when a bootstrap is requested. as a result you can mount a NFS drive, or some other network drive (s3) and use that as a fallback bootstrap source.
- simple configuration simplest configuration is ~ 40 lines between the agent/clients, a few ports and at most two DNS records.
- can work entirely inside of a VPN no need to expose anything to the outside of the network. (requires using ACME DNS configuration)
