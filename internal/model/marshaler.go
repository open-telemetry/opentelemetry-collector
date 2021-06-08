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

// TracesMarshaler marshals pdata.Traces into bytes.
type TracesMarshaler interface {
	Marshal(td pdata.Traces) ([]byte, error)
}

type tracesMarshaler struct {
	encoder    TracesEncoder
	translator FromTracesTranslator
}

// NewTracesMarshaler returns a new TracesMarshaler.
func NewTracesMarshaler(encoder TracesEncoder, translator FromTracesTranslator) TracesMarshaler {
	return &tracesMarshaler{
		encoder:    encoder,
		translator: translator,
	}
}

// Marshal pdata.Traces into bytes. On error []byte is nil.
func (t *tracesMarshaler) Marshal(td pdata.Traces) ([]byte, error) {
	model, err := t.translator.FromTraces(td)
	if err != nil {
		return nil, fmt.Errorf("converting pdata to model failed: %w", err)
	}
	buf, err := t.encoder.EncodeTraces(model)
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %w", err)
	}
	return buf, nil
}

// MetricsMarshaler marshals pdata.Metrics into bytes.
type MetricsMarshaler interface {
	Marshal(td pdata.Metrics) ([]byte, error)
}

type metricsMarshaler struct {
	encoder    MetricsEncoder
	translator FromMetricsTranslator
}

// NewMetricsMarshaler returns a new MetricsMarshaler.
func NewMetricsMarshaler(encoder MetricsEncoder, translator FromMetricsTranslator) MetricsMarshaler {
	return &metricsMarshaler{
		encoder:    encoder,
		translator: translator,
	}
}

// Marshal pdata.Metrics into bytes. On error []byte is nil.
func (t *metricsMarshaler) Marshal(td pdata.Metrics) ([]byte, error) {
	model, err := t.translator.FromMetrics(td)
	if err != nil {
		return nil, fmt.Errorf("converting pdata to model failed: %w", err)
	}
	buf, err := t.encoder.EncodeMetrics(model)
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %w", err)
	}
	return buf, nil
}

// LogsMarshaler marshals pdata.Logs into bytes.
type LogsMarshaler interface {
	Marshal(td pdata.Logs) ([]byte, error)
}

type logsMarshaler struct {
	encoder    LogsEncoder
	translator FromLogsTranslator
}

// NewLogsMarshaler returns a new LogsMarshaler.
func NewLogsMarshaler(encoder LogsEncoder, translator FromLogsTranslator) LogsMarshaler {
	return &logsMarshaler{
		encoder:    encoder,
		translator: translator,
	}
}

// Marshal pdata.Logs into bytes. On error []byte is nil.
func (t *logsMarshaler) Marshal(td pdata.Logs) ([]byte, error) {
	model, err := t.translator.FromLogs(td)
	if err != nil {
		return nil, fmt.Errorf("converting pdata to model failed: %w", err)
	}
	buf, err := t.encoder.EncodeLogs(model)
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %w", err)
	}
	return buf, nil
}
