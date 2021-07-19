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

package otlpgrpc

import (
	"context"

	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/model/internal"
	otlpcollectortrace "go.opentelemetry.io/collector/model/internal/data/protogen/collector/trace/v1"
	"go.opentelemetry.io/collector/model/pdata"
)

// TODO: Consider to add `TracesRequest`. If we add non pdata properties we can add them to the request.

// TracesResponse represents the response for gRPC client/server.
type TracesResponse struct {
	orig *otlpcollectortrace.ExportTraceServiceResponse
}

// NewTracesResponse returns an empty TracesResponse.
func NewTracesResponse() TracesResponse {
	return TracesResponse{orig: &otlpcollectortrace.ExportTraceServiceResponse{}}
}

// TracesClient is the client API for OTLP-GRPC Traces service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type TracesClient interface {
	// Export pdata.Traces to the server.
	//
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(ctx context.Context, in pdata.Traces, opts ...grpc.CallOption) (TracesResponse, error)
}

type tracesClient struct {
	rawClient otlpcollectortrace.TraceServiceClient
}

// NewTracesClient returns a new TracesClient connected using the given connection.
func NewTracesClient(cc *grpc.ClientConn) TracesClient {
	return &tracesClient{rawClient: otlpcollectortrace.NewTraceServiceClient(cc)}
}

// Export implements the TracesClient interface.
func (c *tracesClient) Export(ctx context.Context, in pdata.Traces, opts ...grpc.CallOption) (TracesResponse, error) {
	rsp, err := c.rawClient.Export(ctx, internal.TracesToOtlp(in.InternalRep()), opts...)
	return TracesResponse{orig: rsp}, err
}

// TracesServer is the server API for OTLP gRPC TracesService service.
type TracesServer interface {
	// Export is called every time a new request is received.
	//
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(context.Context, pdata.Traces) (TracesResponse, error)
}

// RegisterTracesServer registers the TracesServer to the grpc.Server.
func RegisterTracesServer(s *grpc.Server, srv TracesServer) {
	otlpcollectortrace.RegisterTraceServiceServer(s, &rawTracesServer{srv: srv})
}

type rawTracesServer struct {
	srv TracesServer
}

func (s rawTracesServer) Export(ctx context.Context, request *otlpcollectortrace.ExportTraceServiceRequest) (*otlpcollectortrace.ExportTraceServiceResponse, error) {
	rsp, err := s.srv.Export(ctx, pdata.TracesFromInternalRep(internal.TracesFromOtlp(request)))
	return rsp.orig, err
}
