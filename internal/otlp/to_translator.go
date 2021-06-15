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
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal"
	otlpcollectorlogs "go.opentelemetry.io/collector/internal/data/protogen/collector/logs/v1"
	otlpcollectormetrics "go.opentelemetry.io/collector/internal/data/protogen/collector/metrics/v1"
	otlpcollectortrace "go.opentelemetry.io/collector/internal/data/protogen/collector/trace/v1"
	"go.opentelemetry.io/collector/internal/model"
)

type toTranslator struct{}

func newToTranslator() *toTranslator {
	return &toTranslator{}
}

func (d *toTranslator) ToLogs(modelData interface{}) (pdata.Logs, error) {
	ld, ok := modelData.(*otlpcollectorlogs.ExportLogsServiceRequest)
	if !ok {
		return pdata.Logs{}, model.NewErrIncompatibleType(&otlpcollectorlogs.ExportLogsServiceRequest{}, modelData)
	}
	return pdata.LogsFromInternalRep(internal.LogsFromOtlp(ld)), nil
}

func (d *toTranslator) ToMetrics(modelData interface{}) (pdata.Metrics, error) {
	ld, ok := modelData.(*otlpcollectormetrics.ExportMetricsServiceRequest)
	if !ok {
		return pdata.Metrics{}, model.NewErrIncompatibleType(&otlpcollectormetrics.ExportMetricsServiceRequest{}, modelData)
	}
	return pdata.MetricsFromInternalRep(internal.MetricsFromOtlp(ld)), nil
}

func (d *toTranslator) ToTraces(modelData interface{}) (pdata.Traces, error) {
	td, ok := modelData.(*otlpcollectortrace.ExportTraceServiceRequest)
	if !ok {
		return pdata.Traces{}, model.NewErrIncompatibleType(&otlpcollectortrace.ExportTraceServiceRequest{}, modelData)
	}
	return pdata.TracesFromInternalRep(internal.TracesFromOtlp(td)), nil
}
