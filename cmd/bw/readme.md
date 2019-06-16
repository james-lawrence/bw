### agent commands
```bash
# generate self signed certificates for the environment
go install github.com/james-lawrence/bw/cmd/...; bwcreds self-signed default localhost 127.0.0.1 127.0.0.2 127.0.0.3 127.0.0.4
go install github.com/james-lawrence/bw/cmd/...; NETWORK=127.0.0.1; bw agent --agent-name="node1" --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:2001 --cluster-bind-raft=$NETWORK:2002 --agent-torrent=${NETWORK}:2003 --cluster=127.0.0.2:2001 --agent-config=".bwagent1/agent.config"
go install github.com/james-lawrence/bw/cmd/...; NETWORK=127.0.0.2; bw agent --agent-name="node2" --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:2001 --cluster-bind-raft=$NETWORK:2002 --agent-torrent=${NETWORK}:2003 --cluster=127.0.0.1:2001 --agent-config=".bwagent2/agent.config"
go install github.com/james-lawrence/bw/cmd/...; NETWORK=127.0.0.3; bw agent --agent-name="node3" --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:2001 --cluster-bind-raft=$NETWORK:2002 --agent-torrent=${NETWORK}:2003 --cluster=127.0.0.1:2001 --agent-config=".bwagent3/agent.config"
go install github.com/james-lawrence/bw/cmd/...; NETWORK=127.0.0.4; bw agent --agent-name="node4" --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:2001 --cluster-bind-raft=$NETWORK:2002 --agent-torrent=${NETWORK}:2003 --cluster=127.0.0.1:2001 --agent-config=".bwagent4/agent.config"
```

### client commands
```bash
go install github.com/james-lawrence/bw/cmd/...; bw deploy
NETWORK=127.0.0.1; bw notify --agent-address=$NETWORK:2000 --agent-config=".bwagent1/agent.config"
```

### notification command
```
bw notify --agent-config=".bwagent1/agent.config" --agent-address=127.0.0.1:2000
```

### getting started
```
bw environment create {workspace} {server-address}
bwcreds vault {workspace} {PKI_PATH} {server-address}
```

```
go install -ldflags '-w -extldflags "-static"' -a github.com/james-lawrence/bw/cmd/...
```
