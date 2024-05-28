// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package debugexporter // import "go.opentelemetry.io/collector/exporter/debugexporter"

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestTracesExporterNoErrors(t *testing.T) {
	lte, err := createTracesExporter(context.Background(), exportertest.NewNopCreateSettings(), createDefaultConfig())
	require.NotNil(t, lte)
	assert.NoError(t, err)

	assert.NoError(t, lte.ConsumeTraces(context.Background(), ptrace.NewTraces()))
	assert.NoError(t, lte.ConsumeTraces(context.Background(), testdata.GenerateTraces(10)))

	assert.NoError(t, lte.Shutdown(context.Background()))
}

func TestMetricsExporterNoErrors(t *testing.T) {
	lme, err := createMetricsExporter(context.Background(), exportertest.NewNopCreateSettings(), createDefaultConfig())
	require.NotNil(t, lme)
	assert.NoError(t, err)

	assert.NoError(t, lme.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.NoError(t, lme.ConsumeMetrics(context.Background(), testdata.GenerateMetricsAllTypes()))
	assert.NoError(t, lme.ConsumeMetrics(context.Background(), testdata.GenerateMetricsAllTypesEmpty()))
	assert.NoError(t, lme.ConsumeMetrics(context.Background(), testdata.GenerateMetricsMetricTypeInvalid()))
	assert.NoError(t, lme.ConsumeMetrics(context.Background(), testdata.GenerateMetrics(10)))

	assert.NoError(t, lme.Shutdown(context.Background()))
}

func TestLogsExporterNoErrors(t *testing.T) {
	lle, err := createLogsExporter(context.Background(), exportertest.NewNopCreateSettings(), createDefaultConfig())
	require.NotNil(t, lle)
	assert.NoError(t, err)

	assert.NoError(t, lle.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.NoError(t, lle.ConsumeLogs(context.Background(), testdata.GenerateLogs(10)))

	assert.NoError(t, lle.Shutdown(context.Background()))
}

func TestExporterErrors(t *testing.T) {
	le := newDebugExporter(zaptest.NewLogger(t), configtelemetry.LevelDetailed)
	require.NotNil(t, le)

	errWant := errors.New("my error")
	le.tracesMarshaler = &errMarshaler{err: errWant}
	le.metricsMarshaler = &errMarshaler{err: errWant}
	le.logsMarshaler = &errMarshaler{err: errWant}
	assert.Equal(t, errWant, le.pushTraces(context.Background(), ptrace.NewTraces()))
	assert.Equal(t, errWant, le.pushMetrics(context.Background(), pmetric.NewMetrics()))
	assert.Equal(t, errWant, le.pushLogs(context.Background(), plog.NewLogs()))
}

type errMarshaler struct {
	err error
}

func (e errMarshaler) MarshalLogs(plog.Logs) ([]byte, error) {
	return nil, e.err
}

func (e errMarshaler) MarshalMetrics(pmetric.Metrics) ([]byte, error) {
	return nil, e.err
}

func (e errMarshaler) MarshalTraces(ptrace.Traces) ([]byte, error) {
	return nil, e.err
}
