### Bearded Wookie configuration management.
secure, fast, reliable, and self contained configuration management.

#### security
bearded-wookie uses TLS to encrypt all data transferred between agents and clients.

#### project goals - how features are decided on and accepted. (no particular order)
- be a secure, resource efficient, and reliable configuration management tool.
- no centralized server.
- no required infrastructure. any additional infrastructure should be optional.
- low operational complexity. a.k.a easy to use and operate.

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

# reset local environment
systemctl --user daemon-reload; rm -rf ~/.config/bearded-wookie/agent{1,2,3,4}/tls && systemctl --user restart bearded-wookie@agent{1,2,3,4}.service && systemctl --user restart bearded-wookie-deploy-notifications@agent1.service && journalctl -f --user-unit bearded-wookie@agent4.service
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
  - bw does have support for remote bootstrapping, it just not required for normal operation.
- builtin support for custom bootstrapping services. this allows you to lose all a majority of your servers and continue to operating.
  - see bootstrap/filesystem.go for an example. the filesystem example allows you to mount NFS to store the last successful deployment.
- Lower costs by not needing additional infrastructure just to support deployments.
- when something does go wrong, bw is easy to repair. just destroy its cache (default /var/cache/bearded-wookie) and reboot the agents.
- simple configuration simplest configuration is ~ 40 lines between the agent/clients, one port (tcp and udp), and DNS record.
