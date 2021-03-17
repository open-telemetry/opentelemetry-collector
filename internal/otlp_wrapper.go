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

package internal

import (
	otlpcollectorlog "go.opentelemetry.io/collector/internal/data/protogen/collector/logs/v1"
	otlpcollectormetrics "go.opentelemetry.io/collector/internal/data/protogen/collector/metrics/v1"
	otlpcollectortrace "go.opentelemetry.io/collector/internal/data/protogen/collector/trace/v1"
)

// MetricsWrapper is an intermediary struct that is declared in an internal package
// as a way to prevent certain functions of pdata.Metrics data type to be callable by
// any code outside of this module.
type MetricsWrapper struct {
	req *otlpcollectormetrics.ExportMetricsServiceRequest
}

func MetricsToOtlp(mw MetricsWrapper) *otlpcollectormetrics.ExportMetricsServiceRequest {
	return mw.req
}

func MetricsFromOtlp(req *otlpcollectormetrics.ExportMetricsServiceRequest) MetricsWrapper {
	return MetricsWrapper{req: req}
}

// TracesWrapper is an intermediary struct that is declared in an internal package
// as a way to prevent certain functions of pdata.Traces data type to be callable by
// any code outside of this module.
type TracesWrapper struct {
	req *otlpcollectortrace.ExportTraceServiceRequest
}

func TracesToOtlp(mw TracesWrapper) *otlpcollectortrace.ExportTraceServiceRequest {
	return mw.req
}

func TracesFromOtlp(req *otlpcollectortrace.ExportTraceServiceRequest) TracesWrapper {
	return TracesWrapper{req: req}
}

// LogsWrapper is an intermediary struct that is declared in an internal package
// as a way to prevent certain functions of pdata.Logs data type to be callable by
// any code outside of this module.
type LogsWrapper struct {
	req *otlpcollectorlog.ExportLogsServiceRequest
}

func LogsToOtlp(l LogsWrapper) *otlpcollectorlog.ExportLogsServiceRequest {
	return l.req
}

func LogsFromOtlp(req *otlpcollectorlog.ExportLogsServiceRequest) LogsWrapper {
	return LogsWrapper{req: req}
}
