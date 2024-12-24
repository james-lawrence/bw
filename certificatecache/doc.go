/*
Package certificatecache provides the functionality for managing and refreshing certificates.

ACME

	Generally only the PrivateKey and Email fields need to be provided and everything will just work.
	Best Practices:
		- PrivateKey should be stored in an environment variable.

	Example Configuration:
		acme:
			email: "soandso@example.com" # for lets encrypt can email you with any issues.
			caurl: "https://127.0.0.1:14000/dir" # defaults to lets encrypt production systems
			key: '${BEARDED_WOOKIE_PRIVATE_KEY}' # note the single quotes.
			port: 2004
			network: "" # network to bind to, defaults to unspecified (0.0.0.0)
*/
package certificatecache
