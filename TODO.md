### reliability
- improve restart command to be reliable when restart nodes; atm it can nuke the node its proxying through immediately.
- improve quorum message listeners reliability. I believe watchers get assigned to incorrect nodes and persist even when the node drops out of quorum.

### implement wasm work
- implement bw/interp/awselb
- - Restart(ctx context.Context, do func(context.Context) error) (err error)
- implement bw/interp/awselb2
- - Restart(ctx context.Context, do func(context.Context) error) (err error)
- implement bw/interp/env
- - its a dynamic package based on runtime data.
- implement bw/interp/envx
- - Boolean(fallback bool, keys ...string) bool
- - String(fallback string, keys ...string) string
- - Duration(time.Duration, keys ...string) time.Duration
- implement bw/interp/shell
- - Lenient(bool) Option
- - Environ(...string) Option
- - Timeout(time.Duration) Option
- - WorkingDir(string) Option
- - TempDir(string) Option
- - Run(ctx context.Context, cmd string, options ...Option) error
- implement os
- - hoping we get this mostly for free with tinygo. but need to deal with chdir, and getting current directory.
- implement log
- - hoping we get this for free with tinygo.
- implement context
- - should get this mostly for free with tinygo. biggest problem is Background() we're we've overridden the default behavior.

### agent connection pool.
- do we still need this?

### nginx serving bearded-wookie on tls port with standard tls server.
```nginx
stream {
	map $ssl_preread_alpn_protocols $proxy {
		~\bacme-tls/1\b 127.0.0.1:2000;
		~\bbw.muxer\b 127.0.0.1:2000;
		default 127.0.0.1:8443;
	}

	server {
		listen 443;
		listen [::]:443;
		proxy_pass $proxy;
		ssl_preread on;
	}
}
```
### useful command for verifying certificates during testing.
openssl s_client -verify_return_error -CAfile ~/.cache/bearded-wookie/agent1/tls/tlsserver.bootstrap.cert -connect bearded-wookie.lan:2000


###
- add --quorum flag to filters.
- 2023/12/06 21:53:00 main.go:98: *errors.withStack - [failed to retrieve info: rpc error: code = Unavailable desc = connection error: desc = "transport: Error while dialing muxer.DialContext failed: bw.agent tcp://10.129.0.97:2000: handshake failed: proxy request failed: ClientError"]
- failed to initiated deploy: rpc error: code = Unavailable desc = error reading from server: EOF
