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
	otlptrace "go.opentelemetry.io/collector/internal/data/protogen/trace/v1"
)

// MetricsWrapper is an intermediary struct that is declared in an internal package
// as a way to prevent certain functions of pdata.Metrics data type to be callable by
// any code outside of this module.
type MetricsWrapper struct {
	req *otlpcollectormetrics.ExportMetricsServiceRequest
}

// MetricsToOtlp internal helper to convert MetricsWrapper to protobuf representation.
func MetricsToOtlp(mw MetricsWrapper) *otlpcollectormetrics.ExportMetricsServiceRequest {
	return mw.req
}

// MetricsFromOtlp internal helper to convert protobuf representation to MetricsWrapper.
func MetricsFromOtlp(req *otlpcollectormetrics.ExportMetricsServiceRequest) MetricsWrapper {
	return MetricsWrapper{req: req}
}

// TracesWrapper is an intermediary struct that is declared in an internal package
// as a way to prevent certain functions of pdata.Traces data type to be callable by
// any code outside of this module.
type TracesWrapper struct {
	req *otlpcollectortrace.ExportTraceServiceRequest
}

// TracesToOtlp internal helper to convert TracesWrapper to protobuf representation.
func TracesToOtlp(mw TracesWrapper) *otlpcollectortrace.ExportTraceServiceRequest {
	return mw.req
}

// TracesFromOtlp internal helper to convert protobuf representation to TracesWrapper.
func TracesFromOtlp(req *otlpcollectortrace.ExportTraceServiceRequest) TracesWrapper {
	return TracesWrapper{req: req}
}

// TracesCompatibilityChanges performs backward compatibility conversion of Span Status code according to
// OTLP specification as we are a new receiver and sender (we are pushing data to the pipelines):
// See https://github.com/open-telemetry/opentelemetry-proto/blob/59c488bfb8fb6d0458ad6425758b70259ff4a2bd/opentelemetry/proto/trace/v1/trace.proto#L239
// See https://github.com/open-telemetry/opentelemetry-proto/blob/59c488bfb8fb6d0458ad6425758b70259ff4a2bd/opentelemetry/proto/trace/v1/trace.proto#L253
func TracesCompatibilityChanges(req *otlpcollectortrace.ExportTraceServiceRequest) {
	for _, rss := range req.ResourceSpans {
		for _, ils := range rss.InstrumentationLibrarySpans {
			for _, span := range ils.Spans {
				switch span.Status.Code {
				case otlptrace.Status_STATUS_CODE_UNSET:
					if span.Status.DeprecatedCode != otlptrace.Status_DEPRECATED_STATUS_CODE_OK {
						span.Status.Code = otlptrace.Status_STATUS_CODE_ERROR
					}
				case otlptrace.Status_STATUS_CODE_OK:
					// If status code is set then overwrites deprecated.
					span.Status.DeprecatedCode = otlptrace.Status_DEPRECATED_STATUS_CODE_OK
				case otlptrace.Status_STATUS_CODE_ERROR:
					span.Status.DeprecatedCode = otlptrace.Status_DEPRECATED_STATUS_CODE_UNKNOWN_ERROR
				}
			}
		}
	}
}

// LogsWrapper is an intermediary struct that is declared in an internal package
// as a way to prevent certain functions of pdata.Logs data type to be callable by
// any code outside of this module.
type LogsWrapper struct {
	req *otlpcollectorlog.ExportLogsServiceRequest
}

// LogsToOtlp internal helper to convert LogsWrapper to protobuf representation.
func LogsToOtlp(l LogsWrapper) *otlpcollectorlog.ExportLogsServiceRequest {
	return l.req
}

// LogsFromOtlp internal helper to convert protobuf representation to LogsWrapper.
func LogsFromOtlp(req *otlpcollectorlog.ExportLogsServiceRequest) LogsWrapper {
	return LogsWrapper{req: req}
}
