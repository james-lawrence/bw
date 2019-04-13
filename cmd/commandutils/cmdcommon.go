package commandutils

import (
	"github.com/alecthomas/kingpin"
	"github.com/james-lawrence/bw"
)

// This file provides a number of common flags/arguments used across bearded-wookie.
// basically to make sure the CLI stays consistent.

// EnvironmentFlag add a environment flag to the provided command.
func EnvironmentFlag(cmd *kingpin.CmdClause) *kingpin.FlagClause {
	return cmd.Flag("environment", "environment to interact with").Default(bw.DefaultEnvironmentName)
}

// EnvironmentArg add a environment arg to the provided command.
func EnvironmentArg(cmd *kingpin.CmdClause) *kingpin.ArgClause {
	return cmd.Arg("environment", "environment to interact with").Default(bw.DefaultEnvironmentName)
}
