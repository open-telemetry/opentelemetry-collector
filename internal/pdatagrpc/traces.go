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

package pdatagrpc

import (
	"context"

	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal"
	otlpcollectortraces "go.opentelemetry.io/collector/internal/data/protogen/collector/trace/v1"
)

// TODO: Consider to add `TracesRequest` and `TracesResponse`. Right now the funcs return interface{},
//  it would be better and future proof to create a TracesResponse empty struct and return that.
//  So if we ever add things in the OTLP response I can deal with that. Similar for request if we add non pdata properties.

// TracesClient is the client API for OTLP-GRPC Traces service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type TracesClient interface {
	// Export pdata.Traces to the server.
	//
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(ctx context.Context, in pdata.Traces, opts ...grpc.CallOption) (interface{}, error)
}

type tracesClient struct {
	rawClient otlpcollectortraces.TraceServiceClient
}

// NewTracesClient returns a new TracesClient connected using the given connection.
func NewTracesClient(cc *grpc.ClientConn) TracesClient {
	return &tracesClient{rawClient: otlpcollectortraces.NewTraceServiceClient(cc)}
}

// Export implements the TracesClient interface.
func (c *tracesClient) Export(ctx context.Context, in pdata.Traces, opts ...grpc.CallOption) (interface{}, error) {
	return c.rawClient.Export(ctx, internal.TracesToOtlp(in.InternalRep()), opts...)
}

// TracesServer is the server API for OTLP gRPC TracesService service.
type TracesServer interface {
	// Export is called every time a new request is received.
	//
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(context.Context, pdata.Traces) (interface{}, error)
}

// RegisterTracesServer registers the TracesServer to the grpc.Server.
func RegisterTracesServer(s *grpc.Server, srv TracesServer) {
	otlpcollectortraces.RegisterTraceServiceServer(s, &rawTracesServer{srv: srv})
}

type rawTracesServer struct {
	srv TracesServer
}

func (s rawTracesServer) Export(ctx context.Context, request *otlpcollectortraces.ExportTraceServiceRequest) (*otlpcollectortraces.ExportTraceServiceResponse, error) {
	_, err := s.srv.Export(ctx, pdata.TracesFromInternalRep(internal.TracesFromOtlp(request)))
	return &otlpcollectortraces.ExportTraceServiceResponse{}, err
}
