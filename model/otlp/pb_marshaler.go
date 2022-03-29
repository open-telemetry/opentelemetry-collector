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
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/model/pdata/logs"
	"go.opentelemetry.io/collector/model/pdata/metrics"
	"go.opentelemetry.io/collector/model/pdata/traces"
)

// NewProtobufTracesMarshaler returns a traces.Marshaler. Marshals to OTLP binary protobuf bytes.
func NewProtobufTracesMarshaler() traces.Marshaler {
	return newPbMarshaler()
}

// NewProtobufMetricsMarshaler returns a metrics.Marshaler. Marshals to OTLP binary protobuf bytes.
func NewProtobufMetricsMarshaler() metrics.Marshaler {
	return newPbMarshaler()
}

// NewProtobufLogsMarshaler returns a logs.Marshaler. Marshals to OTLP binary protobuf bytes.
func NewProtobufLogsMarshaler() logs.Marshaler {
	return newPbMarshaler()
}

// TODO(#3842): Figure out how we want to represent/return *Sizers.
type pbMarshaler struct{}

func newPbMarshaler() *pbMarshaler {
	return &pbMarshaler{}
}

var _ pdata.TracesSizer = (*pbMarshaler)(nil)
var _ pdata.MetricsSizer = (*pbMarshaler)(nil)
var _ pdata.LogsSizer = (*pbMarshaler)(nil)

func (e *pbMarshaler) MarshalLogs(ld logs.Logs) ([]byte, error) {
	return ipdata.LogsToOtlp(ld).Marshal()
}

func (e *pbMarshaler) MarshalMetrics(md metrics.Metrics) ([]byte, error) {
	return ipdata.MetricsToOtlp(md).Marshal()
}

func (e *pbMarshaler) MarshalTraces(td traces.Traces) ([]byte, error) {
	return ipdata.TracesToOtlp(td).Marshal()
}

func (e *pbMarshaler) TracesSize(td traces.Traces) int {
	return ipdata.TracesToOtlp(td).Size()
}

func (e *pbMarshaler) MetricsSize(md metrics.Metrics) int {
	return ipdata.MetricsToOtlp(md).Size()
}

func (e *pbMarshaler) LogsSize(ld logs.Logs) int {
	return ipdata.LogsToOtlp(ld).Size()
}
