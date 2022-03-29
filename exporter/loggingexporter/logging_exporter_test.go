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
package loggingexporter

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/model/pdata/logs"
	"go.opentelemetry.io/collector/model/pdata/metrics"
	"go.opentelemetry.io/collector/model/pdata/traces"
)

func TestLoggingTracesExporterNoErrors(t *testing.T) {
	lte, err := newTracesExporter(&config.ExporterSettings{}, zap.NewNop(), componenttest.NewNopExporterCreateSettings())
	require.NotNil(t, lte)
	assert.NoError(t, err)

	assert.NoError(t, lte.ConsumeTraces(context.Background(), traces.New()))
	assert.NoError(t, lte.ConsumeTraces(context.Background(), testdata.GenerateTracesTwoSpansSameResourceOneDifferent()))

	assert.NoError(t, lte.Shutdown(context.Background()))
}

func TestLoggingMetricsExporterNoErrors(t *testing.T) {
	lme, err := newMetricsExporter(&config.ExporterSettings{}, zap.NewNop(), componenttest.NewNopExporterCreateSettings())
	require.NotNil(t, lme)
	assert.NoError(t, err)

	assert.NoError(t, lme.ConsumeMetrics(context.Background(), metrics.New()))
	assert.NoError(t, lme.ConsumeMetrics(context.Background(), testdata.GeneratMetricsAllTypesWithSampleDatapoints()))
	assert.NoError(t, lme.ConsumeMetrics(context.Background(), testdata.GenerateMetricsAllTypesEmptyDataPoint()))
	assert.NoError(t, lme.ConsumeMetrics(context.Background(), testdata.GenerateMetricsMetricTypeInvalid()))

	assert.NoError(t, lme.Shutdown(context.Background()))
}

func TestLoggingLogsExporterNoErrors(t *testing.T) {
	lle, err := newLogsExporter(&config.ExporterSettings{}, zap.NewNop(), componenttest.NewNopExporterCreateSettings())
	require.NotNil(t, lle)
	assert.NoError(t, err)

	assert.NoError(t, lle.ConsumeLogs(context.Background(), logs.New()))
	assert.NoError(t, lle.ConsumeLogs(context.Background(), testdata.GenerateLogsOneEmptyResourceLogs()))
	assert.NoError(t, lle.ConsumeLogs(context.Background(), testdata.GenerateLogsNoLogRecords()))
	assert.NoError(t, lle.ConsumeLogs(context.Background(), testdata.GenerateLogsOneEmptyLogRecord()))

	assert.NoError(t, lle.Shutdown(context.Background()))
}

func TestLoggingExporterErrors(t *testing.T) {
	le := newLoggingExporter(zaptest.NewLogger(t))
	require.NotNil(t, le)

	errWant := errors.New("my error")
	le.tracesMarshaler = &errMarshaler{err: errWant}
	le.metricsMarshaler = &errMarshaler{err: errWant}
	le.logsMarshaler = &errMarshaler{err: errWant}
	assert.Equal(t, errWant, le.pushTraces(context.Background(), traces.New()))
	assert.Equal(t, errWant, le.pushMetrics(context.Background(), metrics.New()))
	assert.Equal(t, errWant, le.pushLogs(context.Background(), logs.New()))
}

type errMarshaler struct {
	err error
}

func (e errMarshaler) MarshalLogs(logs.Logs) ([]byte, error) {
	return nil, e.err
}

func (e errMarshaler) MarshalMetrics(metrics.Metrics) ([]byte, error) {
	return nil, e.err
}

func (e errMarshaler) MarshalTraces(traces.Traces) ([]byte, error) {
	return nil, e.err
}
