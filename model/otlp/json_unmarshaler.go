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
	"bytes"

	"github.com/gogo/protobuf/jsonpb"

	otlplogs "go.opentelemetry.io/collector/model/internal/data/protogen/logs/v1"
	otlpmetrics "go.opentelemetry.io/collector/model/internal/data/protogen/metrics/v1"
	otlptrace "go.opentelemetry.io/collector/model/internal/data/protogen/trace/v1"
	ipdata "go.opentelemetry.io/collector/model/internal/pdata"
	"go.opentelemetry.io/collector/model/otlpgrpc"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/model/pdata/logs"
	"go.opentelemetry.io/collector/model/pdata/metrics"
	"go.opentelemetry.io/collector/model/pdata/traces"
)

type jsonUnmarshaler struct {
	delegate jsonpb.Unmarshaler
}

// NewJSONTracesUnmarshaler returns a model.TracesUnmarshaler. Unmarshals from OTLP json bytes.
func NewJSONTracesUnmarshaler() pdata.TracesUnmarshaler {
	return newJSONUnmarshaler()
}

// NewJSONMetricsUnmarshaler returns a model.MetricsUnmarshaler. Unmarshals from OTLP json bytes.
func NewJSONMetricsUnmarshaler() pdata.MetricsUnmarshaler {
	return newJSONUnmarshaler()
}

// NewJSONLogsUnmarshaler returns a model.LogsUnmarshaler. Unmarshals from OTLP json bytes.
func NewJSONLogsUnmarshaler() pdata.LogsUnmarshaler {
	return newJSONUnmarshaler()
}

func newJSONUnmarshaler() *jsonUnmarshaler {
	return &jsonUnmarshaler{delegate: jsonpb.Unmarshaler{}}
}

func (d *jsonUnmarshaler) UnmarshalLogs(buf []byte) (logs.Logs, error) {
	ld := &otlplogs.LogsData{}
	if err := d.delegate.Unmarshal(bytes.NewReader(buf), ld); err != nil {
		return logs.Logs{}, err
	}
	otlpgrpc.InstrumentationLibraryLogsToScope(ld.ResourceLogs)
	return ipdata.LogsFromOtlp(ld), nil
}

func (d *jsonUnmarshaler) UnmarshalMetrics(buf []byte) (metrics.Metrics, error) {
	md := &otlpmetrics.MetricsData{}
	if err := d.delegate.Unmarshal(bytes.NewReader(buf), md); err != nil {
		return metrics.Metrics{}, err
	}
	otlpgrpc.InstrumentationLibraryMetricsToScope(md.ResourceMetrics)
	return ipdata.MetricsFromOtlp(md), nil
}

func (d *jsonUnmarshaler) UnmarshalTraces(buf []byte) (traces.Traces, error) {
	td := &otlptrace.TracesData{}
	if err := d.delegate.Unmarshal(bytes.NewReader(buf), td); err != nil {
		return traces.Traces{}, err
	}
	otlpgrpc.InstrumentationLibrarySpansToScope(td.ResourceSpans)
	return ipdata.TracesFromOtlp(td), nil
}
