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

package otlpgrpc // import "go.opentelemetry.io/collector/model/otlpgrpc"

import (
	"bytes"
	"context"

	"google.golang.org/grpc"

	ipdata "go.opentelemetry.io/collector/model/internal"
	otlpcollectormetrics "go.opentelemetry.io/collector/model/internal/data/protogen/collector/metrics/v1"
	otlpmetrics "go.opentelemetry.io/collector/model/internal/data/protogen/metrics/v1"
	"go.opentelemetry.io/collector/model/internal/otlp"
	"go.opentelemetry.io/collector/model/pdata"
)

// MetricsResponse represents the response for gRPC client/server.
type MetricsResponse struct {
	orig *otlpcollectormetrics.ExportMetricsServiceResponse
}

// NewMetricsResponse returns an empty MetricsResponse.
func NewMetricsResponse() MetricsResponse {
	return MetricsResponse{orig: &otlpcollectormetrics.ExportMetricsServiceResponse{}}
}

// MarshalProto marshals MetricsResponse into proto bytes.
func (mr MetricsResponse) MarshalProto() ([]byte, error) {
	return mr.orig.Marshal()
}

// UnmarshalProto unmarshalls MetricsResponse from proto bytes.
func (mr MetricsResponse) UnmarshalProto(data []byte) error {
	return mr.orig.Unmarshal(data)
}

// MarshalJSON marshals MetricsResponse into JSON bytes.
func (mr MetricsResponse) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := jsonMarshaler.Marshal(&buf, mr.orig); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON unmarshalls MetricsResponse from JSON bytes.
func (mr MetricsResponse) UnmarshalJSON(data []byte) error {
	return jsonUnmarshaler.Unmarshal(bytes.NewReader(data), mr.orig)
}

// MetricsRequest represents the response for gRPC client/server.
type MetricsRequest struct {
	orig *otlpcollectormetrics.ExportMetricsServiceRequest
}

// NewMetricsRequest returns an empty MetricsRequest.
func NewMetricsRequest() MetricsRequest {
	return MetricsRequest{orig: &otlpcollectormetrics.ExportMetricsServiceRequest{}}
}

// MarshalProto marshals MetricsRequest into proto bytes.
func (mr MetricsRequest) MarshalProto() ([]byte, error) {
	return mr.orig.Marshal()
}

// UnmarshalProto unmarshalls MetricsRequest from proto bytes.
func (mr MetricsRequest) UnmarshalProto(data []byte) error {
	return mr.orig.Unmarshal(data)
}

// MarshalJSON marshals MetricsRequest into JSON bytes.
func (mr MetricsRequest) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := jsonMarshaler.Marshal(&buf, mr.orig); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON unmarshalls MetricsRequest from JSON bytes.
func (mr MetricsRequest) UnmarshalJSON(data []byte) error {
	if err := jsonUnmarshaler.Unmarshal(bytes.NewReader(data), mr.orig); err != nil {
		return err
	}
	otlp.InstrumentationLibraryMetricsToScope(mr.orig.ResourceMetrics)
	return nil
}

func (mr MetricsRequest) SetMetrics(ld pdata.Metrics) {
	mr.orig.ResourceMetrics = ipdata.MetricsToOtlp(ld).ResourceMetrics
}

func (mr MetricsRequest) Metrics() pdata.Metrics {
	return ipdata.MetricsFromOtlp(&otlpmetrics.MetricsData{ResourceMetrics: mr.orig.ResourceMetrics})
}

// MetricsClient is the client API for OTLP-GRPC Metrics service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type MetricsClient interface {
	// Export pdata.Metrics to the server.
	//
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(ctx context.Context, request MetricsRequest, opts ...grpc.CallOption) (MetricsResponse, error)
}

type metricsClient struct {
	rawClient otlpcollectormetrics.MetricsServiceClient
}

// NewMetricsClient returns a new MetricsClient connected using the given connection.
func NewMetricsClient(cc *grpc.ClientConn) MetricsClient {
	return &metricsClient{rawClient: otlpcollectormetrics.NewMetricsServiceClient(cc)}
}

func (c *metricsClient) Export(ctx context.Context, request MetricsRequest, opts ...grpc.CallOption) (MetricsResponse, error) {
	rsp, err := c.rawClient.Export(ctx, request.orig, opts...)
	return MetricsResponse{orig: rsp}, err
}

// MetricsServer is the server API for OTLP gRPC MetricsService service.
type MetricsServer interface {
	// Export is called every time a new request is received.
	//
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(context.Context, MetricsRequest) (MetricsResponse, error)
}

// RegisterMetricsServer registers the MetricsServer to the grpc.Server.
func RegisterMetricsServer(s *grpc.Server, srv MetricsServer) {
	otlpcollectormetrics.RegisterMetricsServiceServer(s, &rawMetricsServer{srv: srv})
}

type rawMetricsServer struct {
	srv MetricsServer
}

func (s rawMetricsServer) Export(ctx context.Context, request *otlpcollectormetrics.ExportMetricsServiceRequest) (*otlpcollectormetrics.ExportMetricsServiceResponse, error) {
	rsp, err := s.srv.Export(ctx, MetricsRequest{orig: request})
	return rsp.orig, err
}
