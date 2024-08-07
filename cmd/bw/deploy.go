package main

import (
	"net"
	"os"
	"regexp"
	"time"

	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/deploy"
	"github.com/james-lawrence/bw/deployment"
)

type cmdDeploy struct {
	Env      cmdDeployEnvironment `cmd:"" name:"env" aliases:"deploy" help:"deploy to nodes within the cluster of the specified environment"`
	Locally  cmdDeployLocal       `cmd:"" name:"locally" aliases:"local" help:"deploy to the local system"`
	Snapshot cmdDeploySnapshot    `cmd:"" name:"snapshot" help:"generate a deployment archive without uploading it anywhere"`
	Redeploy cmdDeployRedeploy    `cmd:"" name:"archive" help:"redeploy an archive to nodes within the cluster of the specified environment"`
	Cancel   cmdDeployCancel      `cmd:"" name:"cancel" help:"cancel any current deploy"`
}

type DeployCluster struct {
	Insecure    bool             `help:"skip tls verification"`
	Silent      bool             `help:"prevent logs from being generated during a deploy"`
	Debug       bool             `help:"leaves artifacts on the filesystem for debugging"`
	Lenient     bool             `name:"ignore-failures" help:"ignore failed deploys"`
	Canary      bool             `name:"canary" help:"deploy to the canary server" default:"false"`
	Heartbeat   time.Duration    `name:"heartbeat" help:"frequency at which the deploy should emit a heartbeat" default:"10s"`
	Names       []*regexp.Regexp `name:"name" help:"regex to match names against"`
	IPs         []net.IP         `name:"ip" help:"match against the provided IP addresses"`
	Concurrency int64            `name:"concurrency" help:"number of nodes allowed to deploy simultaneously"`
}

type cmdDeployEnvironment struct {
	cmdopts.BeardedWookieEnv
	DeployCluster
}

func (t cmdDeployEnvironment) Run(ctx *cmdopts.Global) error {
	filters := make([]deployment.Filter, 0, len(t.Names))
	for _, n := range t.Names {
		filters = append(filters, deployment.Named(n))
	}

	for _, n := range t.IPs {
		filters = append(filters, deployment.IP(n))
	}

	// need a filter to be present for the canary to work.
	if t.Canary {
		filters = append(filters, deployment.AlwaysMatch)
	}

	return deploy.Into(&deploy.Context{
		Context:     ctx.Context,
		CancelFunc:  ctx.Shutdown,
		WaitGroup:   ctx.Cleanup,
		Verbose:     ctx.Verbosity > 0,
		Environment: t.Environment,
		Concurrency: t.Concurrency,
		Insecure:    t.Insecure,
		Heartbeat:   t.Heartbeat,
		Lenient:     t.Lenient,
		Silent:      t.Silent,
		Canary:      t.Canary,
		Debug:       t.Debug,
		Filter:      deployment.Or(filters...),
		AllowEmpty:  len(filters) == 0,
	})
}

type cmdDeployRedeploy struct {
	DeployCluster
	cmdopts.BeardedWookieEnvRequired
	DeploymentID string `arg:"" name:"deployment-id"`
}

func (t cmdDeployRedeploy) Run(ctx *cmdopts.Global) error {
	filters := make([]deployment.Filter, 0, len(t.Names))
	for _, n := range t.Names {
		filters = append(filters, deployment.Named(n))
	}

	for _, n := range t.IPs {
		filters = append(filters, deployment.IP(n))
	}

	// need a filter to be present for the canary to work.
	if t.Canary {
		filters = append(filters, deployment.AlwaysMatch)
	}

	return deploy.Redeploy(&deploy.Context{
		Context:     ctx.Context,
		CancelFunc:  ctx.Shutdown,
		WaitGroup:   ctx.Cleanup,
		Verbose:     ctx.Verbosity > 0,
		Environment: t.Environment,
		Concurrency: t.Concurrency,
		Insecure:    t.Insecure,
		Lenient:     t.Lenient,
		Heartbeat:   t.Heartbeat,
		Silent:      t.Silent,
		Canary:      t.Canary,
		Debug:       t.Debug,
		Filter:      deployment.Or(filters...),
		AllowEmpty:  len(filters) == 0,
	}, t.DeploymentID)
}

type cmdDeployLocal struct {
	DeployCluster
	cmdopts.BeardedWookieEnv
}

func (t cmdDeployLocal) Run(ctx *cmdopts.Global) error {
	return deploy.Locally(&deploy.Context{
		Context:     ctx.Context,
		CancelFunc:  ctx.Shutdown,
		WaitGroup:   ctx.Cleanup,
		Verbose:     ctx.Verbosity > 0,
		Insecure:    t.Insecure,
		Environment: t.Environment,
		Debug:       t.Debug,
	}, t.Debug)
}

type cmdDeploySnapshot struct {
	cmdopts.BeardedWookieEnv
	snapshotOutput *os.File `name:"output" help:"file to write to. by default archive is written to stdout" short:"o"`
}

func (t cmdDeploySnapshot) Run(ctx *cmdopts.Global) error {
	if t.snapshotOutput == nil {
		t.snapshotOutput = os.Stdout
	}

	return deploy.Snapshot(deploy.Context{
		Context:     ctx.Context,
		CancelFunc:  ctx.Shutdown,
		WaitGroup:   ctx.Cleanup,
		Verbose:     ctx.Verbosity > 0,
		Environment: t.Environment,
	}, t.snapshotOutput)
}

type cmdDeployCancel struct {
	DeployCluster
	cmdopts.BeardedWookieEnv
}

func (t cmdDeployCancel) Run(ctx *cmdopts.Global) error {
	return deploy.Cancel(&deploy.Context{
		Context:     ctx.Context,
		CancelFunc:  ctx.Shutdown,
		WaitGroup:   ctx.Cleanup,
		Verbose:     ctx.Verbosity > 0,
		Environment: t.Environment,
		Insecure:    t.Insecure,
		Debug:       t.Debug,
	})
}
