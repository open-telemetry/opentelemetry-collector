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
	otlpcollectormetrics "go.opentelemetry.io/collector/internal/data/protogen/collector/metrics/v1"
)

// TODO: Consider to add `MetricsRequest` and `MetricsResponse`. Right now the funcs return interface{},
//  it would be better and future proof to create a MetricsResponse empty struct and return that.
//  So if we ever add things in the OTLP response I can deal with that. Similar for request if we add non pdata properties.

// MetricsClient is the client API for OTLP-GRPC Metrics service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type MetricsClient interface {
	// Export pdata.Metrics to the server.
	//
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(ctx context.Context, in pdata.Metrics, opts ...grpc.CallOption) (interface{}, error)
}

type metricsClient struct {
	rawClient otlpcollectormetrics.MetricsServiceClient
}

// NewMetricsClient returns a new MetricsClient connected using the given connection.
func NewMetricsClient(cc *grpc.ClientConn) MetricsClient {
	return &metricsClient{rawClient: otlpcollectormetrics.NewMetricsServiceClient(cc)}
}

func (c *metricsClient) Export(ctx context.Context, in pdata.Metrics, opts ...grpc.CallOption) (interface{}, error) {
	return c.rawClient.Export(ctx, internal.MetricsToOtlp(in.InternalRep()), opts...)
}

// MetricsServer is the server API for OTLP gRPC MetricsService service.
type MetricsServer interface {
	// Export is called every time a new request is received.
	//
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(context.Context, pdata.Metrics) (interface{}, error)
}

// RegisterMetricsServer registers the MetricsServer to the grpc.Server.
func RegisterMetricsServer(s *grpc.Server, srv MetricsServer) {
	otlpcollectormetrics.RegisterMetricsServiceServer(s, &rawMetricsServer{srv: srv})
}

type rawMetricsServer struct {
	srv MetricsServer
}

func (s rawMetricsServer) Export(ctx context.Context, request *otlpcollectormetrics.ExportMetricsServiceRequest) (*otlpcollectormetrics.ExportMetricsServiceResponse, error) {
	_, err := s.srv.Export(ctx, pdata.MetricsFromInternalRep(internal.MetricsFromOtlp(request)))
	return &otlpcollectormetrics.ExportMetricsServiceResponse{}, err
}
