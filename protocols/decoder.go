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
	"go.opentelemetry.io/collector/protocols/bytes"
	"go.opentelemetry.io/collector/protocols/models"
)

type MetricsDecoder struct {
	mod models.MetricsDecoder
	enc bytes.MetricsDecoder
}

type TracesDecoder struct {
	mod models.TracesDecoder
	enc bytes.TracesDecoder
}

type LogsDecoder struct {
	mod models.LogsDecoder
	enc bytes.LogsDecoder
}

// DecodeMetrics decodes bytes to pdata.
func (t *MetricsDecoder) DecodeMetrics(data []byte) (pdata.Metrics, error) {
	model, err := t.enc.DecodeMetrics(data)
	if err != nil {
		return pdata.NewMetrics(), err
	}
	return t.mod.ToMetrics(model)
}

// DecodeTraces decodes bytes to pdata.
func (t *TracesDecoder) DecodeTraces(data []byte) (pdata.Traces, error) {
	model, err := t.enc.DecodeTraces(data)
	if err != nil {
		return pdata.NewTraces(), err
	}
	return t.mod.ToTraces(model)
}

// DecodeLogs decodes bytes to pdata.
func (t *LogsDecoder) DecodeLogs(data []byte) (pdata.Logs, error) {
	model, err := t.enc.DecodeLogs(data)
	if err != nil {
		return pdata.NewLogs(), err
	}
	return t.mod.ToLogs(model)
}