### agent commands
```bash
# generate self signed certificates for the environment
go install github.com/james-lawrence/bw/cmd/...
# use bearded wookie to bootstrap the local development cluster
bw deploy local linux-dev

systemctl --user restart bearded-wookie@agent1.service
systemctl --user restart bearded-wookie@agent2.service
systemctl --user restart bearded-wookie@agent3.service
systemctl --user restart bearded-wookie@agent4.service
```

### client commands
```bash
cd .test && bw deploy
```

### notification command
```
systemctl --user restart bearded-wookie-deploy-notifications@agent1
```

### getting started
```
bw environment create {workspace} {server-address}
bwcreds vault {workspace} {PKI_PATH} {server-address}
```

```
go install -ldflags '-w -extldflags "-static"' -a github.com/james-lawrence/bw/cmd/...
```
