// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ptraceotlp // import "go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"

import (
	"context"

	"google.golang.org/grpc"

	otlpcollectortrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/trace/v1"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
)

// GRPCClient is the client API for OTLP-GRPC Traces service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type GRPCClient interface {
	// Export ptrace.Traces to the server.
	//
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(ctx context.Context, request ExportRequest, opts ...grpc.CallOption) (ExportResponse, error)
}

// NewGRPCClient returns a new GRPCClient connected using the given connection.
func NewGRPCClient(cc *grpc.ClientConn) GRPCClient {
	return &grpcClient{rawClient: otlpcollectortrace.NewTraceServiceClient(cc)}
}

// Deprecated: [v0.63.0]: use NewGRPCClient.
var NewClient = NewGRPCClient

type grpcClient struct {
	rawClient otlpcollectortrace.TraceServiceClient
}

// Export implements the Client interface.
func (c *grpcClient) Export(ctx context.Context, request ExportRequest, opts ...grpc.CallOption) (ExportResponse, error) {
	rsp, err := c.rawClient.Export(ctx, request.orig, opts...)
	return ExportResponse{orig: rsp}, err
}

// GRPCServer is the server API for OTLP gRPC TracesService service.
type GRPCServer interface {
	// Export is called every time a new request is received.
	//
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(context.Context, ExportRequest) (ExportResponse, error)
}

// RegisterGRPCServer registers the GRPCServer to the grpc.Server.
func RegisterGRPCServer(s *grpc.Server, srv GRPCServer) {
	otlpcollectortrace.RegisterTraceServiceServer(s, &rawTracesServer{srv: srv})
}

type rawTracesServer struct {
	srv GRPCServer
}

func (s rawTracesServer) Export(ctx context.Context, request *otlpcollectortrace.ExportTraceServiceRequest) (*otlpcollectortrace.ExportTraceServiceResponse, error) {
	otlp.MigrateTraces(request.ResourceSpans)
	rsp, err := s.srv.Export(ctx, ExportRequest{orig: request})
	return rsp.orig, err
}
