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

type MetricsDecoder interface {
	// ToMetrics converts a data model of another protocol into pdata.
	ToMetrics(src interface{}) (pdata.Metrics, error)
	// Type returns an instance of the model.
	NewModel() interface{}
}

type TracesDecoder interface {
	// ToTraces converts a data model of another protocol into pdata.
	ToTraces(src interface{}) (pdata.Traces, error)
	// NewModel returns an instance of the model.
	NewModel() interface{}
}

type LogsDecoder interface {
	// ToLogs converts a data model of another protocol into pdata.
	ToLogs(src interface{}) (pdata.Logs, error)
	// NewModel returns an instance of the model.
	NewModel() interface{}
}
