# End user Guide - understanding bearded-wookie

bearded-wookie fundamental goal is to get software on servers
in a reliable and low maintenance way. it takes a lot of inspiration
from KISS as in most things are detected from the environment and filesystem.

There were a few guiding lights for bw
- a tool that could support managing developer workstations *and* deployments into productions environments.
  primarily to reduce tooling overhead.
- not require *any* infrastructure for the deployed environments into which you are managing.
- trivial debugging


### getting started

First generate an environment and a workspace for your directives and an environment you want to deploy into.
you 

```bash
bw workspace create --example
bw environment create --name demo
```

These commands will create two directories .bw and .bwconfig/demo the idea here is that the steps for
a given workspace apply to many different environments.

the workspace example that is generated has examples of how to use bw directives to deploy.

### deployment phase .local

this phase is where the artifacts for inclusion into the deployment are gathered
and generated.

### deployment phase .remote

this phase is what actually executes on the remote server.
