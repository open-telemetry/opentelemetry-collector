// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pentityotlp // import "go.opentelemetry.io/collector/pdata/pentity/pentityotlp"

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectorentity "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/entities/v1"
)

// GRPCClient is the client API for OTLP-GRPC Entities service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type GRPCClient interface {
	// Export pentity.Entities to the server.
	//
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(ctx context.Context, request ExportRequest, opts ...grpc.CallOption) (ExportResponse, error)

	// unexported disallow implementation of the GRPCClient.
	unexported()
}

// NewGRPCClient returns a new GRPCClient connected using the given connection.
func NewGRPCClient(cc *grpc.ClientConn) GRPCClient {
	return &grpcClient{rawClient: otlpcollectorentity.NewEntitiesServiceClient(cc)}
}

type grpcClient struct {
	rawClient otlpcollectorentity.EntitiesServiceClient
}

func (c *grpcClient) Export(ctx context.Context, request ExportRequest, opts ...grpc.CallOption) (ExportResponse, error) {
	rsp, err := c.rawClient.Export(ctx, request.orig, opts...)
	if err != nil {
		return ExportResponse{}, err
	}
	state := internal.StateMutable
	return ExportResponse{orig: rsp, state: &state}, err
}

func (c *grpcClient) unexported() {}

// GRPCServer is the server API for OTLP gRPC EntitiesService service.
// Implementations MUST embed UnimplementedGRPCServer.
type GRPCServer interface {
	// Export is called every time a new request is received.
	//
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(context.Context, ExportRequest) (ExportResponse, error)

	// unexported disallow implementation of the GRPCServer.
	unexported()
}

var _ GRPCServer = (*UnimplementedGRPCServer)(nil)

// UnimplementedGRPCServer MUST be embedded to have forward compatible implementations.
type UnimplementedGRPCServer struct{}

func (*UnimplementedGRPCServer) Export(context.Context, ExportRequest) (ExportResponse, error) {
	return ExportResponse{}, status.Errorf(codes.Unimplemented, "method Export not implemented")
}

func (*UnimplementedGRPCServer) unexported() {}

// RegisterGRPCServer registers the Server to the grpc.Server.
func RegisterGRPCServer(s *grpc.Server, srv GRPCServer) {
	otlpcollectorentity.RegisterEntitiesServiceServer(s, &rawEntitiesServer{srv: srv})
}

type rawEntitiesServer struct {
	srv GRPCServer
}

func (s rawEntitiesServer) Export(ctx context.Context, request *otlpcollectorentity.ExportEntitiesServiceRequest) (
	*otlpcollectorentity.ExportEntitiesServiceResponse, error,
) {
	state := internal.StateMutable
	rsp, err := s.srv.Export(ctx, ExportRequest{orig: request, state: &state})
	return rsp.orig, err
}
