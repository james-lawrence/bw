package daemons

//
// import (
// 	"net"
// 	"time"
//
// 	"github.com/james-lawrence/bw/notary"
//
// 	"google.golang.org/grpc"
// 	"google.golang.org/grpc/keepalive"
// )
//
// // Discovery initiates the discovery backend.
// func Discovery(ctx Context, config string) (err error) {
// 	var (
// 		bind   net.Listener
// 		ns     notary.Storage
// 		server *grpc.Server
// 	)
//
// 	keepalive := grpc.KeepaliveParams(keepalive.ServerParameters{
// 		MaxConnectionIdle: 1 * time.Hour,
// 		Time:              1 * time.Minute,
// 		Timeout:           2 * time.Minute,
// 	})
//
// 	if ns, err = notary.NewFromFile(config); err != nil {
// 		return err
// 	}
//
// 	server = grpc.NewServer(grpc.Creds(tlscreds), keepalive)
//
// 	notary.New(ns).Bind(server)
//
// 	ctx.grpc(server, bind)
// 	go func() {
// 		<-ctx.Context.Done()
// 	}()
//
// 	return nil
// }
