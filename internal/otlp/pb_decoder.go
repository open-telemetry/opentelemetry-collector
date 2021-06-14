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
	"go.opentelemetry.io/collector/internal"
	otlpcollectorlogs "go.opentelemetry.io/collector/internal/data/protogen/collector/logs/v1"
	otlpcollectormetrics "go.opentelemetry.io/collector/internal/data/protogen/collector/metrics/v1"
	otlpcollectortrace "go.opentelemetry.io/collector/internal/data/protogen/collector/trace/v1"
)

type pbDecoder struct{}

func newPbDecoder() *pbDecoder {
	return &pbDecoder{}
}

func (d *pbDecoder) DecodeLogs(buf []byte) (interface{}, error) {
	ld := &otlpcollectorlogs.ExportLogsServiceRequest{}
	err := ld.Unmarshal(buf)
	return ld, err
}

func (d *pbDecoder) DecodeMetrics(buf []byte) (interface{}, error) {
	md := &otlpcollectormetrics.ExportMetricsServiceRequest{}
	err := md.Unmarshal(buf)
	return md, err
}

func (d *pbDecoder) DecodeTraces(buf []byte) (interface{}, error) {
	td := &otlpcollectortrace.ExportTraceServiceRequest{}
	err := td.Unmarshal(buf)
	if err == nil {
		internal.TracesCompatibilityChanges(td)
	}
	return td, err
}
