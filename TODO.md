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