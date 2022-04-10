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

	"github.com/gogo/protobuf/jsonpb"
	"google.golang.org/grpc"

	ipdata "go.opentelemetry.io/collector/model/internal"
	otlpcollectorlog "go.opentelemetry.io/collector/model/internal/data/protogen/collector/logs/v1"
	otlplogs "go.opentelemetry.io/collector/model/internal/data/protogen/logs/v1"
	"go.opentelemetry.io/collector/model/internal/otlp"
	"go.opentelemetry.io/collector/model/pdata"
)

var jsonMarshaler = &jsonpb.Marshaler{}
var jsonUnmarshaler = &jsonpb.Unmarshaler{}

// LogsResponse represents the response for gRPC client/server.
type LogsResponse struct {
	orig *otlpcollectorlog.ExportLogsServiceResponse
}

// NewLogsResponse returns an empty LogsResponse.
func NewLogsResponse() LogsResponse {
	return LogsResponse{orig: &otlpcollectorlog.ExportLogsServiceResponse{}}
}

// MarshalProto marshals LogsResponse into proto bytes.
func (lr LogsResponse) MarshalProto() ([]byte, error) {
	return lr.orig.Marshal()
}

// UnmarshalProto unmarshalls LogsResponse from proto bytes.
func (lr LogsResponse) UnmarshalProto(data []byte) error {
	return lr.orig.Unmarshal(data)
}

// MarshalJSON marshals LogsResponse into JSON bytes.
func (lr LogsResponse) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := jsonMarshaler.Marshal(&buf, lr.orig); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON unmarshalls LogsResponse from JSON bytes.
func (lr LogsResponse) UnmarshalJSON(data []byte) error {
	return jsonUnmarshaler.Unmarshal(bytes.NewReader(data), lr.orig)
}

// LogsRequest represents the response for gRPC client/server.
type LogsRequest struct {
	orig *otlpcollectorlog.ExportLogsServiceRequest
}

// NewLogsRequest returns an empty LogsRequest.
func NewLogsRequest() LogsRequest {
	return LogsRequest{orig: &otlpcollectorlog.ExportLogsServiceRequest{}}
}

// MarshalProto marshals LogsRequest into proto bytes.
func (lr LogsRequest) MarshalProto() ([]byte, error) {
	return lr.orig.Marshal()
}

// UnmarshalProto unmarshalls LogsRequest from proto bytes.
func (lr LogsRequest) UnmarshalProto(data []byte) error {
	if err := lr.orig.Unmarshal(data); err != nil {
		return err
	}
	otlp.InstrumentationLibraryLogsToScope(lr.orig.ResourceLogs)
	return nil
}

// MarshalJSON marshals LogsRequest into JSON bytes.
func (lr LogsRequest) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := jsonMarshaler.Marshal(&buf, lr.orig); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON unmarshalls LogsRequest from JSON bytes.
func (lr LogsRequest) UnmarshalJSON(data []byte) error {
	if err := jsonUnmarshaler.Unmarshal(bytes.NewReader(data), lr.orig); err != nil {
		return err
	}
	otlp.InstrumentationLibraryLogsToScope(lr.orig.ResourceLogs)
	return nil
}

func (lr LogsRequest) SetLogs(ld pdata.Logs) {
	lr.orig.ResourceLogs = ipdata.LogsToOtlp(ld).ResourceLogs
}

func (lr LogsRequest) Logs() pdata.Logs {
	return ipdata.LogsFromOtlp(&otlplogs.LogsData{ResourceLogs: lr.orig.ResourceLogs})
}

// LogsClient is the client API for OTLP-GRPC Logs service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type LogsClient interface {
	// Export pdata.Logs to the server.
	//
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(ctx context.Context, request LogsRequest, opts ...grpc.CallOption) (LogsResponse, error)
}

type logsClient struct {
	rawClient otlpcollectorlog.LogsServiceClient
}

// NewLogsClient returns a new LogsClient connected using the given connection.
func NewLogsClient(cc *grpc.ClientConn) LogsClient {
	return &logsClient{rawClient: otlpcollectorlog.NewLogsServiceClient(cc)}
}

func (c *logsClient) Export(ctx context.Context, request LogsRequest, opts ...grpc.CallOption) (LogsResponse, error) {
	rsp, err := c.rawClient.Export(ctx, request.orig, opts...)
	return LogsResponse{orig: rsp}, err
}

// LogsServer is the server API for OTLP gRPC LogsService service.
type LogsServer interface {
	// Export is called every time a new request is received.
	//
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(context.Context, LogsRequest) (LogsResponse, error)
}

// RegisterLogsServer registers the LogsServer to the grpc.Server.
func RegisterLogsServer(s *grpc.Server, srv LogsServer) {
	otlpcollectorlog.RegisterLogsServiceServer(s, &rawLogsServer{srv: srv})
}

type rawLogsServer struct {
	srv LogsServer
}

func (s rawLogsServer) Export(ctx context.Context, request *otlpcollectorlog.ExportLogsServiceRequest) (*otlpcollectorlog.ExportLogsServiceResponse, error) {
	rsp, err := s.srv.Export(ctx, LogsRequest{orig: request})
	return rsp.orig, err
}
