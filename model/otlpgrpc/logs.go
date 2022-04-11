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
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
)

// LogsResponse represents the response for gRPC client/server.
// Deprecated: [v0.49.0] Use plogotlp.Response instead.
type LogsResponse = plogotlp.Response

// NewLogsResponse returns an empty LogsResponse.
// Deprecated: [v0.49.0] Use plogotlp.NewResponse instead.
var NewLogsResponse = plogotlp.NewResponse

// LogsRequest represents the response for gRPC client/server.
// Deprecated: [v0.49.0] Use plogotlp.Request instead.
type LogsRequest = plogotlp.Request

// NewLogsRequest returns an empty LogsRequest.
// Deprecated: [v0.49.0] Use plogotlp.NewRequest instead.
var NewLogsRequest = plogotlp.NewRequest

// LogsClient is the client API for OTLP-GRPC Logs service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
// Deprecated: [v0.49.0] Use plogotlp.Client instead.
type LogsClient = plogotlp.Client

// NewLogsClient returns a new LogsClient connected using the given connection.
// Deprecated: [v0.49.0] Use plogotlp.NewClient instead.
var NewLogsClient = plogotlp.NewClient

// LogsServer is the server API for OTLP gRPC LogsService service.
// Deprecated: [v0.49.0] Use plogotlp.Server instead.
type LogsServer = plogotlp.Server

// RegisterLogsServer registers the LogsServer to the grpc.Server.
// Deprecated: [v0.49.0] Use plogotlp.RegisterServer instead.
var RegisterLogsServer = plogotlp.RegisterServer
