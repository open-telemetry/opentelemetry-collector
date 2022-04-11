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

package otlp // import "go.opentelemetry.io/collector/model/otlp"

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// NewProtobufTracesUnmarshaler returns a model.TracesUnmarshaler. Unmarshals from OTLP binary protobuf bytes.
// Deprecated: [v0.49.0] Use ptrace.NewProtoUnmarshaler instead.
var NewProtobufTracesUnmarshaler = ptrace.NewProtoUnmarshaler

// NewProtobufMetricsUnmarshaler returns a model.MetricsUnmarshaler. Unmarshals from OTLP binary protobuf bytes.
// Deprecated: [v0.49.0] Use pmetric.NewProtoUnmarshaler instead.
var NewProtobufMetricsUnmarshaler = pmetric.NewProtoUnmarshaler

// NewProtobufLogsUnmarshaler returns a model.LogsUnmarshaler. Unmarshals from OTLP binary protobuf bytes.
// Deprecated: [v0.49.0] Use plog.NewProtoUnmarshaler instead.
var NewProtobufLogsUnmarshaler = plog.NewProtoUnmarshaler
