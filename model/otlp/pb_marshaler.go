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
	ipdata "go.opentelemetry.io/collector/model/internal/pdata"
	"go.opentelemetry.io/collector/model/plog"
	"go.opentelemetry.io/collector/model/pmetric"
	"go.opentelemetry.io/collector/model/ptrace"
)

// NewProtobufTracesMarshaler returns a ptrace.TracesMarshaler. Marshals to OTLP binary protobuf bytes.
func NewProtobufTracesMarshaler() ptrace.TracesMarshaler {
	return newPbMarshaler()
}

// NewProtobufMetricsMarshaler returns a pmetric.MetricsMarshaler. Marshals to OTLP binary protobuf bytes.
func NewProtobufMetricsMarshaler() pmetric.MetricsMarshaler {
	return newPbMarshaler()
}

// NewProtobufLogsMarshaler returns a plog.LogsMarshaler. Marshals to OTLP binary protobuf bytes.
func NewProtobufLogsMarshaler() plog.LogsMarshaler {
	return newPbMarshaler()
}

// TODO(#3842): Figure out how we want to represent/return *Sizers.
type pbMarshaler struct{}

func newPbMarshaler() *pbMarshaler {
	return &pbMarshaler{}
}

var _ ptrace.TracesSizer = (*pbMarshaler)(nil)
var _ pmetric.MetricsSizer = (*pbMarshaler)(nil)
var _ plog.LogsSizer = (*pbMarshaler)(nil)

func (e *pbMarshaler) MarshalLogs(ld plog.Logs) ([]byte, error) {
	return ipdata.LogsToOtlp(ld).Marshal()
}

func (e *pbMarshaler) MarshalMetrics(md pmetric.Metrics) ([]byte, error) {
	return ipdata.MetricsToOtlp(md).Marshal()
}

func (e *pbMarshaler) MarshalTraces(td ptrace.Traces) ([]byte, error) {
	return ipdata.TracesToOtlp(td).Marshal()
}

func (e *pbMarshaler) TracesSize(td ptrace.Traces) int {
	return ipdata.TracesToOtlp(td).Size()
}

func (e *pbMarshaler) MetricsSize(md pmetric.Metrics) int {
	return ipdata.MetricsToOtlp(md).Size()
}

func (e *pbMarshaler) LogsSize(ld plog.Logs) int {
	return ipdata.LogsToOtlp(ld).Size()
}
