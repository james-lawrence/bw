### agent commands
```bash
NETWORK=127.0.0.1; ./bin/bw agent --agent-name="node1" --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:2001 --cluster-bind-raft=$NETWORK:2002 --agent-torrent=${NETWORK}:2003 --cluster=127.0.0.2:2001 --agent-config=".bwagent1/agent.config"
NETWORK=127.0.0.2; ./bin/bw agent --agent-name="node2" --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:2001 --cluster-bind-raft=$NETWORK:2002 --agent-torrent=${NETWORK}:2003 --cluster=127.0.0.1:2001 --agent-config=".bwagent2/agent.config"
NETWORK=127.0.0.3; ./bin/bw agent --agent-name="node3" --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:2001 --cluster-bind-raft=$NETWORK:2002 --agent-torrent=${NETWORK}:2003 --cluster=127.0.0.1:2001 --agent-config=".bwagent3/agent.config"
NETWORK=127.0.0.4; ./bin/bw agent --agent-name="node4" --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:2001 --cluster-bind-raft=$NETWORK:2002 --agent-torrent=${NETWORK}:2003 --cluster=127.0.0.1:2001 --agent-config=".bwagent4/agent.config"
```

### client commands
```bash
./bin/bw deploy
./bin/bw notify --agent-address=$NETWORK:2000
```

### notification command
```
./bin/bw notify --agent-config=".bwagent1/agent.config" --agent-address=127.0.0.1:2000
```
