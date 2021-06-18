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

package model

import "go.opentelemetry.io/collector/consumer/pdata"

// ToMetricsTranslator is an interface to translate a protocol-specific data model into pdata.Traces.
type ToMetricsTranslator interface {
	// ToMetrics translates a protocol-specific data model into pdata.Metrics.
	// If the error is not nil, the returned pdata.Metrics cannot be used.
	ToMetrics(src interface{}) (pdata.Metrics, error)
}

// ToTracesTranslator is an interface to translate a protocol-specific data model into pdata.Traces.
type ToTracesTranslator interface {
	// ToTraces translates a protocol-specific data model into pdata.Traces.
	// If the error is not nil, the returned pdata.Traces cannot be used.
	ToTraces(src interface{}) (pdata.Traces, error)
}

// ToLogsTranslator is an interface to translate a protocol-specific data model into pdata.Traces.
type ToLogsTranslator interface {
	// ToLogs translates a protocol-specific data model into pdata.Logs.
	// If the error is not nil, the returned pdata.Logs cannot be used.
	ToLogs(src interface{}) (pdata.Logs, error)
}
