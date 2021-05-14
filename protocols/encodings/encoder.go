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

package encodings

import "go.opentelemetry.io/collector/consumer/pdata"

// MetricsEncoder converts pdata to another type.
type MetricsEncoder interface {
	MarshalMetrics(md pdata.Metrics) ([]byte, error)
}

// TracesEncoder converts pdata to another type.
type TracesEncoder interface {
	MarshalTraces(md pdata.Traces) ([]byte, error)
}

// LogsEncoder converts pdata to another type.
type LogsEncoder interface {
	MarshalLogs(md pdata.Logs) ([]byte, error)
}
