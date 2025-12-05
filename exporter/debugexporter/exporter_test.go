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

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter/debugexporter/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestTracesNoErrors(t *testing.T) {
	for _, tc := range createTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			lte, err := createTraces(context.Background(), exportertest.NewNopSettings(metadata.Type), tc.config)
			require.NotNil(t, lte)
			assert.NoError(t, err)

			assert.NoError(t, lte.ConsumeTraces(context.Background(), ptrace.NewTraces()))
			assert.NoError(t, lte.ConsumeTraces(context.Background(), testdata.GenerateTraces(10)))

			assert.NoError(t, lte.Shutdown(context.Background()))
		})
	}
}

func TestMetricsNoErrors(t *testing.T) {
	for _, tc := range createTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			lme, err := createMetrics(context.Background(), exportertest.NewNopSettings(metadata.Type), tc.config)
			require.NotNil(t, lme)
			assert.NoError(t, err)

			assert.NoError(t, lme.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
			assert.NoError(t, lme.ConsumeMetrics(context.Background(), testdata.GenerateMetricsAllTypes()))
			assert.NoError(t, lme.ConsumeMetrics(context.Background(), testdata.GenerateMetricsAllTypesEmpty()))
			assert.NoError(t, lme.ConsumeMetrics(context.Background(), testdata.GenerateMetricsMetricTypeInvalid()))
			assert.NoError(t, lme.ConsumeMetrics(context.Background(), testdata.GenerateMetrics(10)))

			assert.NoError(t, lme.Shutdown(context.Background()))
		})
	}
}

func TestLogsNoErrors(t *testing.T) {
	for _, tc := range createTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			lle, err := createLogs(context.Background(), exportertest.NewNopSettings(metadata.Type), tc.config)
			require.NotNil(t, lle)
			assert.NoError(t, err)

			assert.NoError(t, lle.ConsumeLogs(context.Background(), plog.NewLogs()))
			assert.NoError(t, lle.ConsumeLogs(context.Background(), testdata.GenerateLogs(10)))

			assert.NoError(t, lle.Shutdown(context.Background()))
		})
	}
}

func TestProfilesNoErrors(t *testing.T) {
	for _, tc := range createTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			lle, err := createProfiles(context.Background(), exportertest.NewNopSettings(metadata.Type), tc.config)
			require.NotNil(t, lle)
			assert.NoError(t, err)

			assert.NoError(t, lle.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
			assert.NoError(t, lle.ConsumeProfiles(context.Background(), testdata.GenerateProfiles(10)))

			assert.NoError(t, lle.Shutdown(context.Background()))
		})
	}
}

func TestErrors(t *testing.T) {
	le := newDebugExporter(zaptest.NewLogger(t), configtelemetry.LevelDetailed)
	require.NotNil(t, le)

	errWant := errors.New("my error")
	le.tracesMarshaler = &errMarshaler{err: errWant}
	le.metricsMarshaler = &errMarshaler{err: errWant}
	le.logsMarshaler = &errMarshaler{err: errWant}
	le.profilesMarshaler = &errMarshaler{err: errWant}
	assert.Equal(t, errWant, le.pushTraces(context.Background(), ptrace.NewTraces()))
	assert.Equal(t, errWant, le.pushMetrics(context.Background(), pmetric.NewMetrics()))
	assert.Equal(t, errWant, le.pushLogs(context.Background(), plog.NewLogs()))
	assert.Equal(t, errWant, le.pushProfiles(context.Background(), pprofile.NewProfiles()))
}

type testCase struct {
	name   string
	config *Config
}

func createTestCases() []testCase {
	return []testCase{
		{
			name: "default config",
			config: func() *Config {
				c := createDefaultConfig().(*Config)
				c.QueueConfig = configoptional.Some(exporterhelper.NewDefaultQueueConfig())
				c.QueueConfig.Get().QueueSize = 10
				return c
			}(),
		},
		{
			name: "don't use internal logger",
			config: func() *Config {
				cfg := createDefaultConfig().(*Config)
				cfg.QueueConfig = configoptional.Some(exporterhelper.NewDefaultQueueConfig())
				cfg.QueueConfig.Get().QueueSize = 10
				cfg.UseInternalLogger = false
				return cfg
			}(),
		},
	}
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

func (e errMarshaler) MarshalProfiles(pprofile.Profiles) ([]byte, error) {
	return nil, e.err
}
