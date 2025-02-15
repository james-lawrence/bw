// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.2
// source: discovery.proto

package discovery

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	Discovery_Quorum_FullMethodName = "/discovery.Discovery/Quorum"
	Discovery_Agents_FullMethodName = "/discovery.Discovery/Agents"
)

// DiscoveryClient is the client API for Discovery service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Discovery service provides information about the cluster. typically this is
// used for establishing connections with the quorum nodes, which are
// responsible for persisting data needed by the cluster.
type DiscoveryClient interface {
	Quorum(ctx context.Context, in *QuorumRequest, opts ...grpc.CallOption) (*QuorumResponse, error)
	Agents(ctx context.Context, in *AgentsRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[AgentsResponse], error)
}

type discoveryClient struct {
	cc grpc.ClientConnInterface
}

func NewDiscoveryClient(cc grpc.ClientConnInterface) DiscoveryClient {
	return &discoveryClient{cc}
}

func (c *discoveryClient) Quorum(ctx context.Context, in *QuorumRequest, opts ...grpc.CallOption) (*QuorumResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(QuorumResponse)
	err := c.cc.Invoke(ctx, Discovery_Quorum_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *discoveryClient) Agents(ctx context.Context, in *AgentsRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[AgentsResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Discovery_ServiceDesc.Streams[0], Discovery_Agents_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[AgentsRequest, AgentsResponse]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Discovery_AgentsClient = grpc.ServerStreamingClient[AgentsResponse]

// DiscoveryServer is the server API for Discovery service.
// All implementations must embed UnimplementedDiscoveryServer
// for forward compatibility.
//
// Discovery service provides information about the cluster. typically this is
// used for establishing connections with the quorum nodes, which are
// responsible for persisting data needed by the cluster.
type DiscoveryServer interface {
	Quorum(context.Context, *QuorumRequest) (*QuorumResponse, error)
	Agents(*AgentsRequest, grpc.ServerStreamingServer[AgentsResponse]) error
	mustEmbedUnimplementedDiscoveryServer()
}

// UnimplementedDiscoveryServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedDiscoveryServer struct{}

func (UnimplementedDiscoveryServer) Quorum(context.Context, *QuorumRequest) (*QuorumResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Quorum not implemented")
}
func (UnimplementedDiscoveryServer) Agents(*AgentsRequest, grpc.ServerStreamingServer[AgentsResponse]) error {
	return status.Errorf(codes.Unimplemented, "method Agents not implemented")
}
func (UnimplementedDiscoveryServer) mustEmbedUnimplementedDiscoveryServer() {}
func (UnimplementedDiscoveryServer) testEmbeddedByValue()                   {}

// UnsafeDiscoveryServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DiscoveryServer will
// result in compilation errors.
type UnsafeDiscoveryServer interface {
	mustEmbedUnimplementedDiscoveryServer()
}

func RegisterDiscoveryServer(s grpc.ServiceRegistrar, srv DiscoveryServer) {
	// If the following call pancis, it indicates UnimplementedDiscoveryServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Discovery_ServiceDesc, srv)
}

func _Discovery_Quorum_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QuorumRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DiscoveryServer).Quorum(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Discovery_Quorum_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DiscoveryServer).Quorum(ctx, req.(*QuorumRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Discovery_Agents_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(AgentsRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(DiscoveryServer).Agents(m, &grpc.GenericServerStream[AgentsRequest, AgentsResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Discovery_AgentsServer = grpc.ServerStreamingServer[AgentsResponse]

// Discovery_ServiceDesc is the grpc.ServiceDesc for Discovery service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Discovery_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "discovery.Discovery",
	HandlerType: (*DiscoveryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Quorum",
			Handler:    _Discovery_Quorum_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Agents",
			Handler:       _Discovery_Agents_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "discovery.proto",
}

const (
	Authority_Check_FullMethodName = "/discovery.Authority/Check"
)

// AuthorityClient is the client API for Authority service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Authority service provides methods for ensure TLS credentials are correct.
type AuthorityClient interface {
	Check(ctx context.Context, in *CheckRequest, opts ...grpc.CallOption) (*CheckResponse, error)
}

type authorityClient struct {
	cc grpc.ClientConnInterface
}

func NewAuthorityClient(cc grpc.ClientConnInterface) AuthorityClient {
	return &authorityClient{cc}
}

func (c *authorityClient) Check(ctx context.Context, in *CheckRequest, opts ...grpc.CallOption) (*CheckResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CheckResponse)
	err := c.cc.Invoke(ctx, Authority_Check_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AuthorityServer is the server API for Authority service.
// All implementations must embed UnimplementedAuthorityServer
// for forward compatibility.
//
// Authority service provides methods for ensure TLS credentials are correct.
type AuthorityServer interface {
	Check(context.Context, *CheckRequest) (*CheckResponse, error)
	mustEmbedUnimplementedAuthorityServer()
}

// UnimplementedAuthorityServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedAuthorityServer struct{}

func (UnimplementedAuthorityServer) Check(context.Context, *CheckRequest) (*CheckResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Check not implemented")
}
func (UnimplementedAuthorityServer) mustEmbedUnimplementedAuthorityServer() {}
func (UnimplementedAuthorityServer) testEmbeddedByValue()                   {}

// UnsafeAuthorityServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AuthorityServer will
// result in compilation errors.
type UnsafeAuthorityServer interface {
	mustEmbedUnimplementedAuthorityServer()
}

func RegisterAuthorityServer(s grpc.ServiceRegistrar, srv AuthorityServer) {
	// If the following call pancis, it indicates UnimplementedAuthorityServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Authority_ServiceDesc, srv)
}

func _Authority_Check_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CheckRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthorityServer).Check(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Authority_Check_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthorityServer).Check(ctx, req.(*CheckRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Authority_ServiceDesc is the grpc.ServiceDesc for Authority service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Authority_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "discovery.Authority",
	HandlerType: (*AuthorityServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Check",
			Handler:    _Authority_Check_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "discovery.proto",
}
