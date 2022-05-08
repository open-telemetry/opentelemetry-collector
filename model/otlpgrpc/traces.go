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
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
)

// TracesResponse represents the response for gRPC client/server.
// Deprecated: [v0.49.0] Use ptraceotlp.Response instead.
type TracesResponse = ptraceotlp.Response

// NewTracesResponse returns an empty TracesResponse.
// Deprecated: [v0.49.0] Use ptraceotlp.NewResponse instead.
var NewTracesResponse = ptraceotlp.NewResponse

// TracesRequest represents the response for gRPC client/server.
// Deprecated: [v0.49.0] Use ptraceotlp.Request instead.
type TracesRequest = ptraceotlp.Request

// NewTracesRequest returns an empty TracesRequest.
// Deprecated: [v0.49.0] Use ptraceotlp.NewRequest instead.
var NewTracesRequest = ptraceotlp.NewRequest

// TracesClient is the client API for OTLP-GRPC Traces service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
// Deprecated: [v0.49.0] Use ptraceotlp.Client instead.
type TracesClient = ptraceotlp.Client

// NewTracesClient returns a new TracesClient connected using the given connection.
// Deprecated: [v0.49.0] Use ptraceotlp.NewClient instead.
var NewTracesClient = ptraceotlp.NewClient

// TracesServer is the server API for OTLP gRPC TracesService service.
// Deprecated: [v0.49.0] Use ptraceotlp.Server instead.
type TracesServer = ptraceotlp.Server

// RegisterTracesServer registers the TracesServer to the grpc.Server.
// Deprecated: [v0.49.0] Use ptraceotlp.RegisterServer instead.
var RegisterTracesServer = ptraceotlp.RegisterServer
