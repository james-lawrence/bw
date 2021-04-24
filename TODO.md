### muxer integration goals
bearded wookie has grown and become more and more complicated, mostly at the network/authorization
areas. as a result the code has grown somewhat hard to maintain. putting a connection multiplexer
in front of the socket should resolve a huge chunk of that complexity.
- reduce number of sockets required by default down to 1.
- allow for support of quic and other future IP streaming protocols.

### muxer pending work
- [x] move grpc services to muxer
- [x] move gossip protocol to muxer, memberlist has a Transport config option
- [x] move raft protocol to muxer, care may have to be taken that it doesn't destroy the socket during shutdown.
- [x] move torrent protocol to muxer, this should (hopefully) be straight forward due to the work done on the custom torrent library.
- [x] implement merge state for agent notary key
- [x] switch to using signed requests everywhere.
- [x] fix tests

### muxer nice to haves
- [ ] automatic tunneling to internal services (quorum, deployments) so we don't have to implement additional proxy grpc services.
- [ ] client authorization at connection using notary service.
- [ ] decouple autocert (ACME, Vault, custom) implementations from primary daemon.
- [ ] fix raft overlay by decoupling the API and writing better tests.

### investigation done
- attempted to use libp2p. was overly complicated and not flexible enough. enough though in theory it could have supported everything.


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