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

package translation

import "go.opentelemetry.io/collector/consumer/pdata"

type MetricsEncoder interface {
	// FromMetrics converts pdata to data model.
	FromMetrics(md pdata.Metrics, out interface{}) error
	// NewModel returns an instance of the model.
	NewModel() interface{}
}

type TracesEncoder interface {
	// FromTraces converts pdata to data model.
	FromTraces(md pdata.Traces, out interface{}) error
	// NewModel returns an instance of the model.
	NewModel() interface{}
}

type LogsEncoder interface {
	// FromLogs converts pdata to data model.
	FromLogs(md pdata.Logs, out interface{}) error
	// NewModel returns an instance of the model.
	NewModel() interface{}
}
