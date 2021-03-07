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
- [x] move torrent protocol to muxer, this should (hopefully) be straight forward due to the work I did on the custom torrent library.
- [ ] implement merge state for the TLSCA instead of the shitty 1 hour polling.
- [ ] implement merge state for agent notary key
- [ ] switch to using signed requests everywhere.
- [ ] fix tests

### muxer nice to haves
- [ ] automatic tunneling to internal services (quorum, deployments) so we don't have to implement additional proxy grpc services.
- [ ] client authorization at connection using notary service.
- [ ] decouple autocert (ACME, Vault, custom) implementations from primary daemon.
- [ ] fix raft overlay by decoupling the API and writing better tests.

### investigation done
- attempted to use libp2p. was overly complicated and not flexible enough. enough though in theory it could have supported everything.


Mar 27 07:22:45 dambli bearded-wookie-agent2[336758]: [AGENT - agent2] 2021/03/27 07:22:45 libp2psocket.go:21: torrentx.Socket Dial 127.0.0.1:2005