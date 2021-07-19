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
	otlpcollectorlog "go.opentelemetry.io/collector/model/internal/data/protogen/collector/logs/v1"
	"go.opentelemetry.io/collector/model/pdata"
)

// TODO: Consider to add `LogsRequest`. If we add non pdata properties we can add them to the request.

// LogsResponse represents the response for gRPC client/server.
type LogsResponse struct {
	orig *otlpcollectorlog.ExportLogsServiceResponse
}

// NewLogsResponse returns an empty LogsResponse.
func NewLogsResponse() LogsResponse {
	return LogsResponse{orig: &otlpcollectorlog.ExportLogsServiceResponse{}}
}

// LogsClient is the client API for OTLP-GRPC Logs service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type LogsClient interface {
	// Export pdata.Logs to the server.
	//
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(ctx context.Context, in pdata.Logs, opts ...grpc.CallOption) (LogsResponse, error)
}

type logsClient struct {
	rawClient otlpcollectorlog.LogsServiceClient
}

// NewLogsClient returns a new LogsClient connected using the given connection.
func NewLogsClient(cc *grpc.ClientConn) LogsClient {
	return &logsClient{rawClient: otlpcollectorlog.NewLogsServiceClient(cc)}
}

func (c *logsClient) Export(ctx context.Context, in pdata.Logs, opts ...grpc.CallOption) (LogsResponse, error) {
	rsp, err := c.rawClient.Export(ctx, internal.LogsToOtlp(in.InternalRep()), opts...)
	return LogsResponse{orig: rsp}, err
}

// LogsServer is the server API for OTLP gRPC LogsService service.
type LogsServer interface {
	// Export is called every time a new request is received.
	//
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(context.Context, pdata.Logs) (LogsResponse, error)
}

// RegisterLogsServer registers the LogsServer to the grpc.Server.
func RegisterLogsServer(s *grpc.Server, srv LogsServer) {
	otlpcollectorlog.RegisterLogsServiceServer(s, &rawLogsServer{srv: srv})
}

type rawLogsServer struct {
	srv LogsServer
}

func (s rawLogsServer) Export(ctx context.Context, request *otlpcollectorlog.ExportLogsServiceRequest) (*otlpcollectorlog.ExportLogsServiceResponse, error) {
	rsp, err := s.srv.Export(ctx, pdata.LogsFromInternalRep(internal.LogsFromOtlp(request)))
	return rsp.orig, err
}
