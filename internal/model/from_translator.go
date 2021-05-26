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

// FromMetricsTranslator interface to translate pdata.Metrics to protocol-specific data model.
type FromMetricsTranslator interface {
	// FromMetrics translates pdata.Metrics to protocol-specific data model.
	FromMetrics(md pdata.Metrics) (interface{}, error)
}

// FromTracesTranslator interface to translate pdata.Traces to protocol-specific data model.
type FromTracesTranslator interface {
	// FromTraces translates pdata.Traces to protocol-specific data model.
	FromTraces(td pdata.Traces) (interface{}, error)
}

// FromLogsTranslator interface to translate pdata.Logs to protocol-specific data model.
type FromLogsTranslator interface {
	// FromLogs translates pdata.Logs to protocol-specific data model.
	FromLogs(ld pdata.Logs) (interface{}, error)
}
