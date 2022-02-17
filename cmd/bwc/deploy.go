package main

import (
	"net"
	"os"
	"regexp"

	"github.com/james-lawrence/bw/cmd/bwc/cmdopts"
	"github.com/james-lawrence/bw/cmd/deploy"
	"github.com/james-lawrence/bw/deployment"
)

type deployCmd struct {
	Cluster  cmdDeployEnvironment `cmd:"" name:"env" aliases:"deploy"`
	Locally  cmdDeployLocal       `cmd:"" name:"locally"`
	Snapshot cmdDeploySnapshot    `cmd:"" name:"snapshot"`
	Redeploy cmdDeployRedeploy    `cmd:"" name:"redeploy" aliases:"archive"`
	Cancel   cmdDeployCancel      `cmd:"" name:"cancel"`
}

type DeployCluster struct {
	Insecure    bool             `help:"skip tls verification"`
	Silent      bool             `help:"prevent logs from being generated during a deploy"`
	Lenient     bool             `name:"ignore-failures" help:"ignore failed deploys"`
	Canary      bool             `name:"canary" help:"deploy to the canary server" default:"false"`
	Names       []*regexp.Regexp `name:"name" help:"regex to match names against"`
	IPs         []net.IP         `name:"ip" help:"match against the provided IP addresses"`
	Concurrency int64            `name:"concurrency" help:"number of nodes allowed to deploy simultaneously"`
}

type cmdDeployEnvironment struct {
	BeardedWookieEnv
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
		Context:        ctx.Context,
		CancelFunc:     ctx.Shutdown,
		WaitGroup:      ctx.Cleanup,
		Environment:    t.Environment,
		Concurrency:    t.Concurrency,
		Insecure:       t.Insecure,
		IgnoreFailures: t.Lenient,
		SilenceLogs:    t.Silent,
		Canary:         t.Canary,
		Filter:         deployment.Or(filters...),
		AllowEmpty:     len(filters) == 0,
	})
}

type cmdDeployRedeploy struct {
	DeployCluster
	BeardedWookieEnvRequired
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
		Context:        ctx.Context,
		CancelFunc:     ctx.Shutdown,
		WaitGroup:      ctx.Cleanup,
		Environment:    t.Environment,
		Concurrency:    t.Concurrency,
		Insecure:       t.Insecure,
		IgnoreFailures: t.Lenient,
		SilenceLogs:    t.Silent,
		Canary:         t.Canary,
		Filter:         deployment.Or(filters...),
		AllowEmpty:     len(filters) == 0,
	}, t.DeploymentID)
}

type cmdDeployLocal struct {
	BeardedWookieEnv
	Debug bool `help:"leaves artifacts on the filesystem for debugging"`
}

func (t cmdDeployLocal) Run(ctx *cmdopts.Global) error {
	return deploy.Locally(&deploy.Context{
		Context:     ctx.Context,
		CancelFunc:  ctx.Shutdown,
		WaitGroup:   ctx.Cleanup,
		Environment: t.Environment,
	}, t.Debug)
}

type cmdDeploySnapshot struct {
	BeardedWookieEnv
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
		Environment: t.Environment,
	}, t.snapshotOutput)
}

type cmdDeployCancel struct {
	BeardedWookieEnv
}

func (t cmdDeployCancel) Run(ctx *cmdopts.Global) error {
	return deploy.Cancel(&deploy.Context{
		Context:     ctx.Context,
		CancelFunc:  ctx.Shutdown,
		WaitGroup:   ctx.Cleanup,
		Environment: t.Environment,
	})
}
