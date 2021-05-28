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

import (
	"fmt"

	"go.opentelemetry.io/collector/consumer/pdata"
)

// TracesUnmarshaler unmarshalls bytes into pdata.Traces.
type TracesUnmarshaler interface {
	Unmarshal(buf []byte) (pdata.Traces, error)
}

type tracesUnmarshaler struct {
	encoder    TracesDecoder
	translator ToTracesTranslator
}

// NewTracesUnmarshaler returns a new TracesUnmarshaler.
func NewTracesUnmarshaler(encoder TracesDecoder, translator ToTracesTranslator) TracesUnmarshaler {
	return &tracesUnmarshaler{
		encoder:    encoder,
		translator: translator,
	}
}

// Unmarshal bytes into pdata.Traces. On error pdata.Traces is invalid.
func (t *tracesUnmarshaler) Unmarshal(buf []byte) (pdata.Traces, error) {
	model, err := t.encoder.DecodeTraces(buf)
	if err != nil {
		return pdata.Traces{}, fmt.Errorf("unmarshal failed: %w", err)
	}
	td, err := t.translator.ToTraces(model)
	if err != nil {
		return pdata.Traces{}, fmt.Errorf("converting model to pdata failed: %w", err)
	}
	return td, nil
}

// MetricsUnmarshaler unmarshalls bytes into pdata.Metrics.
type MetricsUnmarshaler interface {
	Unmarshal(buf []byte) (pdata.Metrics, error)
}

type metricsUnmarshaler struct {
	encoder    MetricsDecoder
	translator ToMetricsTranslator
}

// NewMetricsUnmarshaler returns a new MetricsUnmarshaler.
func NewMetricsUnmarshaler(encoder MetricsDecoder, translator ToMetricsTranslator) MetricsUnmarshaler {
	return &metricsUnmarshaler{
		encoder:    encoder,
		translator: translator,
	}
}

// Unmarshal bytes into pdata.Metrics. On error pdata.Metrics is invalid.
func (t *metricsUnmarshaler) Unmarshal(buf []byte) (pdata.Metrics, error) {
	model, err := t.encoder.DecodeMetrics(buf)
	if err != nil {
		return pdata.Metrics{}, fmt.Errorf("unmarshal failed: %w", err)
	}
	td, err := t.translator.ToMetrics(model)
	if err != nil {
		return pdata.Metrics{}, fmt.Errorf("converting model to pdata failed: %w", err)
	}
	return td, nil
}

// LogsUnmarshaler unmarshalls bytes into pdata.Logs.
type LogsUnmarshaler interface {
	Unmarshal(buf []byte) (pdata.Logs, error)
}

type logsUnmarshaler struct {
	encoder    LogsDecoder
	translator ToLogsTranslator
}

// NewLogsUnmarshaler returns a new LogsUnmarshaler.
func NewLogsUnmarshaler(encoder LogsDecoder, translator ToLogsTranslator) LogsUnmarshaler {
	return &logsUnmarshaler{
		encoder:    encoder,
		translator: translator,
	}
}

// Unmarshal bytes into pdata.Logs. On error pdata.Logs is invalid.
func (t *logsUnmarshaler) Unmarshal(buf []byte) (pdata.Logs, error) {
	model, err := t.encoder.DecodeLogs(buf)
	if err != nil {
		return pdata.Logs{}, fmt.Errorf("unmarshal failed: %w", err)
	}
	td, err := t.translator.ToLogs(model)
	if err != nil {
		return pdata.Logs{}, fmt.Errorf("converting model to pdata failed: %w", err)
	}
	return td, nil
}
