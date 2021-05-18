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

package protocols

import (
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/protocols/models"
	"go.opentelemetry.io/collector/protocols/bytes"
)

type MetricsEncoder struct {
	mod models.MetricsEncoder
	enc bytes.MetricsEncoder
}

type TracesEncoder struct {
	mod models.TracesEncoder
	enc bytes.TracesEncoder
}

type LogsEncoder struct {
	mod models.LogsEncoder
	enc bytes.LogsEncoder
}

// EncodeMetrics encodes pdata to bytes.
func (t *MetricsEncoder) EncodeMetrics(td pdata.Metrics) ([]byte, error) {
	out := t.mod.Type()
	if err := t.mod.FromMetrics(td, &out); err != nil {
		return nil, err
	}
	return t.enc.EncodeMetrics(out)
}

// EncodeTraces encodes pdata to bytes.
func (t *TracesEncoder) EncodeTraces(td pdata.Traces) ([]byte, error) {
	out := t.mod.Type()
	if err := t.mod.FromTraces(td, &out); err != nil {
		return nil, err
	}
	return t.enc.EncodeTraces(out)
}

// EncodeLogs encodes pdata to bytes.
func (t *LogsEncoder) EncodeLogs(td pdata.Logs) ([]byte, error) {
	out := t.mod.Type()
	if err := t.mod.FromLogs(td, &out); err != nil {
		return nil, err
	}
	return t.enc.EncodeLogs(out)
}
