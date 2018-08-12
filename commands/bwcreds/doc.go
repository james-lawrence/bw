// Copyright James Lawrence
// All Rights Reserved

// bwcreds provides functionality to initialize TLS credentials for clients and agents.
// self-signed certificate generation.
// vault PKI: see https://www.vaultproject.io/docs/secrets/pki/index.html
//
// Security Concerns
// self signed certificates should generally be secure but use at your own risk.
//
// Vault PKI currently uses the issue endpoint, which transfers the private key
// for the certificate over the wire, if HTTPS isn't used for vault then this isn't
// secure, future versions will use the sign endpoint which fixes this issue.

// Example Self Signed Certificate commands:
//  bwcreds self-signed {environment} {common-name} {hosts...}
//  bwcreds self-signed default
//  bwcreds self-signed default example.com *.example.com 127.0.0.1 127.0.0.2
//  bwcreds self-signed default example.com foo.example.com
//
// Example Vault PKI
//  bwcreds vault {environment} {vault-issue-path} {common-name}
//  bwcreds vault default bwcreds vault default pki/issue/dev-role example.com
package main
