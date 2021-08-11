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

package otlp

import (
	"go.opentelemetry.io/collector/model/internal"
	"go.opentelemetry.io/collector/model/pdata"
)

// NewProtobufTracesMarshaler returns a model.TracesMarshaler. Marshals to OTLP binary protobuf bytes.
func NewProtobufTracesMarshaler() pdata.TracesMarshaler {
	return newPbMarshaler()
}

// NewProtobufMetricsMarshaler returns a model.MetricsMarshaler. Marshals to OTLP binary protobuf bytes.
func NewProtobufMetricsMarshaler() pdata.MetricsMarshaler {
	return newPbMarshaler()
}

// NewProtobufLogsMarshaler returns a model.LogsMarshaler. Marshals to OTLP binary protobuf bytes.
func NewProtobufLogsMarshaler() pdata.LogsMarshaler {
	return newPbMarshaler()
}

func NewProtobufTracesSizer() pdata.TracesSizer {
	return newPbMarshaler()
}

func NewProtobufMetricsSizer() pdata.MetricsSizer {
	return newPbMarshaler()
}

func NewProtobufLogsSizer() pdata.LogsSizer {
	return newPbMarshaler()
}

type pbMarshaler struct{}

func newPbMarshaler() *pbMarshaler {
	return &pbMarshaler{}
}

var _ pdata.TracesSizer = (*pbMarshaler)(nil)
var _ pdata.MetricsSizer = (*pbMarshaler)(nil)
var _ pdata.LogsSizer = (*pbMarshaler)(nil)

func (e *pbMarshaler) MarshalLogs(ld pdata.Logs) ([]byte, error) {
	return internal.LogsToOtlp(ld.InternalRep()).Marshal()
}

func (e *pbMarshaler) MarshalMetrics(md pdata.Metrics) ([]byte, error) {
	return internal.MetricsToOtlp(md.InternalRep()).Marshal()
}

func (e *pbMarshaler) MarshalTraces(td pdata.Traces) ([]byte, error) {
	return internal.TracesToOtlp(td.InternalRep()).Marshal()
}

func (e *pbMarshaler) TracesSize(td pdata.Traces) int {
	return internal.TracesToOtlp(td.InternalRep()).Size()
}

func (e *pbMarshaler) MetricsSize(md pdata.Metrics) int {
	return internal.MetricsToOtlp(md.InternalRep()).Size()
}

func (e *pbMarshaler) LogsSize(ld pdata.Logs) int {
	return internal.LogsToOtlp(ld.InternalRep()).Size()
}
