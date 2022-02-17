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
