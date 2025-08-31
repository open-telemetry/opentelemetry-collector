// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package debugexporter // import "go.opentelemetry.io/collector/exporter/debugexporter"

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter/debugexporter/internal"
	"go.opentelemetry.io/collector/exporter/debugexporter/internal/metadata"
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
			lle, err := createLogs(context.Background(), exportertest.NewNopSettings(metadata.Type), createDefaultConfig())
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
			lle, err := createProfiles(context.Background(), exportertest.NewNopSettings(metadata.Type), createDefaultConfig())
			require.NotNil(t, lle)
			assert.NoError(t, err)

			assert.NoError(t, lle.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
			assert.NoError(t, lle.ConsumeProfiles(context.Background(), testdata.GenerateProfiles(10)))

			assert.NoError(t, lle.Shutdown(context.Background()))
		})
	}
}

func TestErrors(t *testing.T) {
	le := newDebugExporter(zaptest.NewLogger(t), configtelemetry.LevelDetailed, internal.OutputConfig{})
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

func TestResourceAttributesFilter(t *testing.T) {
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	observedLogger := zap.New(observedZapCore)
	exporterWithMultipleAttributesConfig := newDebugExporter(
		observedLogger,
		configtelemetry.LevelDetailed,
		internal.OutputConfig{
			Resource: internal.ResourceOutputConfig{
				AttributesOutputConfig: internal.Attributes{
					Include: []string{"attribute.b", "attribute.c"},
					Enabled: true,
				},
				Enabled: true,
			},
			Record: internal.RecordOutputConfig{
				Enabled: true,
				Attributes: internal.Attributes{
					Include: []string{"attribute.b", "attribute.c"},
					Enabled: true,
				},
			},
			Scope: internal.ScopeOutputConfig{
				Enabled: true,
				Attributes: internal.Attributes{
					Include: []string{"attribute.b", "attribute.c"},
					Enabled: true,
				},
			},
		})
	exporterWithoutAttributesConfig := newDebugExporter(observedLogger, configtelemetry.LevelDetailed, internal.OutputConfig{
		Resource: internal.ResourceOutputConfig{
			Enabled: true,
			AttributesOutputConfig: internal.Attributes{
				Enabled: true,
			},
		},
		Scope: internal.ScopeOutputConfig{
			Enabled: true,
		},
		Record: internal.RecordOutputConfig{
			Enabled: true,
		},
	})

	testCases := []struct {
		name           string
		generateEntity func() any
		pushMethod     func(_ context.Context, _ any) error
		shouldFound    int
		foundCondition func(_ string) bool
	}{
		{
			name: "no attributes filter",
			generateEntity: func() any {
				traces := ptrace.NewTraces()
				attributes := traces.ResourceSpans().AppendEmpty().Resource().Attributes()
				attributes.PutStr("attribute.a", "bbb")
				attributes.PutStr("attribute.b", "bbb")
				attributes.PutStr("attribute.c", "ccc")
				return traces
			},
			pushMethod: func(ctx context.Context, obj any) error {
				return exporterWithoutAttributesConfig.pushTraces(ctx, obj.(ptrace.Traces))
			},
			shouldFound: 3,
			foundCondition: func(m string) bool {
				return strings.Contains(m, "attribute.a: Str(bbb)") || strings.Contains(m, "attribute.b: Str(bbb)") || strings.Contains(m, "attribute.c: Str(ccc)")
			},
		},
		{
			name: "traces with attributes in spans",
			generateEntity: func() any {
				traces := ptrace.NewTraces()
				rs := traces.ResourceSpans().AppendEmpty()
				attributes := rs.Resource().Attributes()
				attributes.PutStr("attribute.a", "bbb")
				attributes.PutStr("attribute.b", "bbb")
				attributes.PutStr("attribute.c", "ccc")
				ss := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
				spanAttributes := ss.Attributes()
				spanAttributes.PutStr("attribute.b", "value.span")
				return traces
			},
			pushMethod: func(ctx context.Context, obj any) error {
				return exporterWithMultipleAttributesConfig.pushTraces(ctx, obj.(ptrace.Traces))
			},
			shouldFound: 3,
			foundCondition: func(m string) bool {
				return strings.Contains(m, "attribute.a: Str(bbb)") || strings.Contains(m, "attribute.b: Str(bbb)") || strings.Contains(m, "attribute.c: Str(ccc)") || strings.Contains(m, "attribute.b: Str(value.span)")
			},
		},
		{
			name: "metrics with attributes",
			generateEntity: func() any {
				metrics := pmetric.NewMetrics()
				rm := metrics.ResourceMetrics().AppendEmpty()
				attributes := rm.Resource().Attributes()
				attributes.PutStr("attribute.a", "bbb")
				attributes.PutStr("attribute.b", "bbb")
				attributes.PutStr("attribute.c", "ccc")
				return metrics
			},
			pushMethod: func(ctx context.Context, obj any) error {
				return exporterWithMultipleAttributesConfig.pushMetrics(ctx, obj.(pmetric.Metrics))
			},
			shouldFound: 2,
			foundCondition: func(m string) bool {
				return strings.Contains(m, "attribute.a: Str(bbb)") || strings.Contains(m, "attribute.b: Str(bbb)") || strings.Contains(m, "attribute.c: Str(ccc)")
			},
		},
		{
			name: "logs with attributes",
			generateEntity: func() any {
				logs := plog.NewLogs()
				rl := logs.ResourceLogs().AppendEmpty()
				attributes := rl.Resource().Attributes()
				attributes.PutStr("attribute.a", "bbb")
				attributes.PutStr("attribute.b", "bbb")
				attributes.PutStr("attribute.c", "ccc")
				log := rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				log.Attributes().PutStr("attribute.b", "log.value")
				return logs
			},
			pushMethod: func(ctx context.Context, obj any) error {
				return exporterWithMultipleAttributesConfig.pushLogs(ctx, obj.(plog.Logs))
			},
			shouldFound: 2,
			foundCondition: func(m string) bool {
				return strings.Contains(m, "attribute.a: Str(bbb)") || strings.Contains(m, "attribute.b: Str(bbb)") || strings.Contains(m, "attribute.c: Str(ccc)") || strings.Contains(m, "attribute.b: Str(logs.value)")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entity := tc.generateEntity()
			require.NoError(t, tc.pushMethod(context.Background(), entity))
			found := 0
			for _, entry := range observedLogs.TakeAll() {
				lines := strings.Split(entry.Message, "\n")
				for _, line := range lines {
					if tc.foundCondition(line) {
						found++
					}
				}
			}
			assert.Equal(t, tc.shouldFound, found)
		})
	}
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
				return createDefaultConfig().(*Config)
			}(),
		},
		{
			name: "don't use internal logger",
			config: func() *Config {
				cfg := createDefaultConfig().(*Config)
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
