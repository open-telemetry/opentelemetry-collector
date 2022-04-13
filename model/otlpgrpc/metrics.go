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
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
)

// MetricsResponse represents the response for gRPC client/server.
// Deprecated: [v0.49.0] Use pmetricotlp.Response instead.
type MetricsResponse = pmetricotlp.Response

// NewMetricsResponse returns an empty MetricsResponse.
// Deprecated: [v0.49.0] Use pmetricotlp.NewResponse instead.
var NewMetricsResponse = pmetricotlp.NewResponse

// MetricsRequest represents the response for gRPC client/server.
// Deprecated: [v0.49.0] Use pmetricotlp.Request instead.
type MetricsRequest = pmetricotlp.Request

// NewMetricsRequest returns an empty MetricsRequest.
// Deprecated: [v0.49.0] Use pmetricotlp.NewRequest instead.
var NewMetricsRequest = pmetricotlp.NewRequest

// MetricsClient is the client API for OTLP-GRPC Metrics service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
// Deprecated: [v0.49.0] Use pmetricotlp.Client instead.
type MetricsClient = pmetricotlp.Client

// NewMetricsClient returns a new MetricsClient connected using the given connection.
// Deprecated: [v0.49.0] Use pmetricotlp.NewClient instead.
var NewMetricsClient = pmetricotlp.NewClient

// MetricsServer is the server API for OTLP gRPC MetricsService service.
// Deprecated: [v0.49.0] Use pmetricotlp.Server instead.
type MetricsServer = pmetricotlp.Server

// RegisterMetricsServer registers the MetricsServer to the grpc.Server.
// Deprecated: [v0.49.0] Use pmetricotlp.RegisterServer instead.
var RegisterMetricsServer = pmetricotlp.RegisterServer
