package main

import "github.com/james-lawrence/bw/cmd/bw/agentcmd"

type cmdBootstrap struct {
	Env agentcmd.CmdBootstrapEnv `cmd:"" name:"env" help:"detect the cloud environment (aws/gcloud) of the host and write the peering configuration to the agent environment file"`
}
