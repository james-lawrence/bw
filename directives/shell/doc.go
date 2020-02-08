// Copyright James Lawrence
// All Rights Reserved

// Package shell provides the ability to execute shell commands.
//
// the shell package provides a few helpful variables that can be accessed
// by shell commands from variable substitution and environment variables.
// value       | environment variable      | variable substitution
// hostname    | BW_ENVIRONMENT_HOST       | %H
// machine ID  | BW_ENVIRONMENT_MACHINE_ID | %m
// domain name | BW_ENVIRONMENT_DOMAIN     | %d
// FQDN        | BW_ENVIRONMENT_FQDN       | %f
// username    | BW_ENVIRONMENT_USERNAME   | %u
// user uid    | BW_ENVIRONMENT_USERID     | %U
// home dir    | BW_ENVIRONMENT_USERHOME   | %h
// working dir | BW_ENVIRONMENT_ROOT       | %bwroot
//
// to escape variable substitution you can use %% to escape the % sign.
package shell
