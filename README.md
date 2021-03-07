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

### quick start - local development (assumes linux).
```bash
# install bearded wookie
go install github.com/james-lawrence/bw/cmd/...

# initialize user credentials.
bw me init

# use bearded wookie to bootstrap the local development cluster
bw deploy local linux-dev

systemctl --user restart bearded-wookie@agent1.service
systemctl --user restart bearded-wookie@agent2.service
systemctl --user restart bearded-wookie@agent3.service
systemctl --user restart bearded-wookie@agent4.service

pushd .test && bw deploy && popd

# TODO
systemctl --user daemon-reload; rm -rf ~/.config/bearded-wookie/agent4/tls ~/.cache/bearded-wookie/agent4/cached.certs && systemctl --user restart bearded-wookie@agent4.service && journalctl -f --user-unit bearded-wookie@agent4.service
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
 - `bw environment create {name} {address}`  
 - `bw workspace bootstrap` creates a deployment workspace. this is a directory + skeleton.  
 - `bw deploy {environment}` deploy to the specified environment  
 - `bw deploy --ip='127.0.0.1' {environment}` deploy to the servers that match the given filters. 
 - `bw deploy --canary {environment}` deploy to a single consistent server. 
 - `bw deploy archive {environment} {deploymentID}` redeploy a previously uploaded archive  
 - `bw deploy archive --ip='127.0.0.1' {environment} {deploymentID}` filter a redeploy to specific servers  
 - `bw info check {address}:{port}` checks if the cluster is reachable.  

commands only available from rpc endpoint (should only be accessed from inside a secure network):  
 - `bw info {environment}` display information and receive events about the environment.  

### architecture overview
bearded-wookie is built off 4 main protocols.
- SWIM - a gossip protocol for discovering peers and agent health checks.
- RAFT - consensus algorithm for shared state, e.g.) current deployment.
- Bit torrent - for archive transfer between servers.
- ACME protocol - for bootstrapping TLS.

by using these 4 protocols bearded-wookie avoids needing a centralized server, while remaining durable.
it also makes bearded-wookie easy to operate.

Benefits of bearded-wookie:
- can work entirely inside of a VPN; no need to expose anything outside of the network. (requires using ACME DNS challenge)
- No need for remote storage, your servers already have a copy of the latest deploy, no need for a backup unless you literally lose all of the leader nodes and a majority of your entire cluster simultaneously - and honestly if that happens you have bigger issues. (note: BW does support a remote backup for bootstrapping, its just not required for normal operation)
- Lower costs by not needing additional infrastructure just to support deployments.
- when something does go wrong, BW is easy to repair. just destroy its cache (default /var/cache/bearded-wookie/deploys) and reboot the agents.
- supports single box deployments - canary deploys can be incredibly useful.
- builtin support for custom bootstrapping - see bootstrap/filesystem.go for an example. the filesystem archives the latest successful deploy to a directory and returns that result when a bootstrap is requested. as a result you can mount a NFS drive, or some other network drive (s3) and use that as a fallback bootstrap source.
- simple configuration simplest configuration is ~ 40 lines between the agent/clients, a few ports and two DNS records.
