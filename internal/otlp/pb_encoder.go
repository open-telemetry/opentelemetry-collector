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
	otlpcollectorlogs "go.opentelemetry.io/collector/internal/data/protogen/collector/logs/v1"
	otlpcollectormetrics "go.opentelemetry.io/collector/internal/data/protogen/collector/metrics/v1"
	otlpcollectortrace "go.opentelemetry.io/collector/internal/data/protogen/collector/trace/v1"
	"go.opentelemetry.io/collector/model/pdata"
)

type pbEncoder struct{}

func newPbEncoder() *pbEncoder {
	return &pbEncoder{}
}

func (e *pbEncoder) EncodeLogs(modelData interface{}) ([]byte, error) {
	ld, ok := modelData.(*otlpcollectorlogs.ExportLogsServiceRequest)
	if !ok {
		return nil, pdata.NewErrIncompatibleType(&otlpcollectorlogs.ExportLogsServiceRequest{}, modelData)
	}
	return ld.Marshal()
}

func (e *pbEncoder) EncodeMetrics(modelData interface{}) ([]byte, error) {
	md, ok := modelData.(*otlpcollectormetrics.ExportMetricsServiceRequest)
	if !ok {
		return nil, pdata.NewErrIncompatibleType(&otlpcollectormetrics.ExportMetricsServiceRequest{}, modelData)
	}
	return md.Marshal()
}

func (e *pbEncoder) EncodeTraces(modelData interface{}) ([]byte, error) {
	td, ok := modelData.(*otlpcollectortrace.ExportTraceServiceRequest)
	if !ok {
		return nil, pdata.NewErrIncompatibleType(&otlpcollectortrace.ExportTraceServiceRequest{}, modelData)
	}
	return td.Marshal()
}
