// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricSdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	traceSdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/service/internal/componentattribute"
	"go.opentelemetry.io/collector/service/internal/metadata"
)

func findScopeAttributesField(context []zap.Field) ([]attribute.KeyValue, bool) {
	for _, field := range context {
		scope, ok := componentattribute.ExtractLogScopeAttributes(field)
		if ok {
			return scope, true
		}
	}
	return nil, false
}

func attributeSetJSON(t *testing.T, set attribute.Set) string {
	scopeBuf, err := json.Marshal(set.MarshalLog())
	require.NoError(t, err)
	return string(scopeBuf)
}

func getLogScopeAndFields(t *testing.T, logObs *observer.ObservedLogs) (string, string) {
	logs := logObs.TakeAll()
	require.Len(t, logs, 1)
	log := logs[0]
	require.Equal(t, "test", log.Message)

	scope, ok := findScopeAttributesField(log.Context)
	require.True(t, ok, "Failed to find ScopeAttributesField field")
	scopeStr := attributeSetJSON(t, attribute.NewSet(scope...))

	enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{})
	fieldsBuf, err := enc.EncodeEntry(log.Entry, log.Context)
	require.NoError(t, err)
	fieldsStr := strings.TrimSuffix(fieldsBuf.String(), "\n")

	return scopeStr, fieldsStr
}

func getSpanScope(t *testing.T, spanObs *tracetest.InMemoryExporter) string {
	spans := spanObs.GetSpans().Snapshots()
	spanObs.Reset()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "test", span.Name())
	return attributeSetJSON(t, span.InstrumentationScope().Attributes)
}

func getMetricScope(t *testing.T, metricObs *metricSdk.ManualReader) string {
	rm := metricdata.ResourceMetrics{}
	err := metricObs.Collect(t.Context(), &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	return attributeSetJSON(t, rm.ScopeMetrics[0].Scope.Attributes)
}

type TestResults struct {
	LogScope    string
	LogFields   string
	SpanScope   string
	MetricScope string
}

func getScopes(t *testing.T, tswa component.TelemetrySettings, logObs *observer.ObservedLogs, spanObs *tracetest.InMemoryExporter, metricObs *metricSdk.ManualReader) TestResults {
	// Create new tracer, meter, and metric instrument
	tracer := tswa.TracerProvider.Tracer("test", trace.WithInstrumentationAttributes(attribute.String("after", "val")))
	meter := tswa.MeterProvider.Meter("test", metric.WithInstrumentationAttributes(attribute.String("after", "val")))
	gauge, err := meter.Int64Gauge("test")
	require.NoError(t, err)

	// Emit a log, a span, and a metric point
	tswa.Logger.Info("test", zap.String("manual", "val"))
	logScope, logFields := getLogScopeAndFields(t, logObs)

	_, span := tracer.Start(t.Context(), "test")
	span.End()

	gauge.Record(t.Context(), 1)

	// Check resulting scope attributes
	return TestResults{
		LogScope:    logScope,
		LogFields:   logFields,
		SpanScope:   getSpanScope(t, spanObs),
		MetricScope: getMetricScope(t, metricObs),
	}
}

type tracerProviderWrapper struct {
	trace.TracerProvider
}

func testTelemetryWithAttributes(t *testing.T, useTraceSdk bool) {
	prevState := metadata.TelemetryNewPipelineTelemetryFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.TelemetryNewPipelineTelemetryFeatureGate.ID(), true))
	defer func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.TelemetryNewPipelineTelemetryFeatureGate.ID(), prevState))
	}()

	// Setup mock TelemetrySettings
	core, logObs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	logger = logger.With(zap.String("before", "val"))

	spanObs := tracetest.NewInMemoryExporter()
	var tracerProvider trace.TracerProvider = traceSdk.NewTracerProvider(traceSdk.WithSpanProcessor(traceSdk.NewSimpleSpanProcessor(spanObs)))
	if !useTraceSdk {
		tracerProvider = tracerProviderWrapper{TracerProvider: tracerProvider}
	}

	// Use delta temporality so points from the first step are no longer exported in the second step
	metricObs := metricSdk.NewManualReader(metricSdk.WithTemporalitySelector(func(metricSdk.InstrumentKind) metricdata.Temporality {
		return metricdata.DeltaTemporality
	}))
	meterProvider := metricSdk.NewMeterProvider(metricSdk.WithReader(metricObs))

	ts := component.TelemetrySettings{
		Logger:         logger,
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
	}

	// Inject attributes
	tswa := componentattribute.TelemetrySettingsWithAttributes(ts, attribute.NewSet(
		attribute.String("injected1", "val"),
		attribute.String("injected2", "val"),
	))

	// Check that SDK-only methods are accessible through Unwrap
	wrapped, ok := tswa.TracerProvider.(interface {
		Unwrap() trace.TracerProvider
	})
	if assert.True(t, ok) {
		_, ok := wrapped.Unwrap().(interface {
			ForceFlush(ctx context.Context) error
		})
		assert.Equal(t, useTraceSdk, ok)
	}

	// Add extra log attribute
	tswa.Logger = tswa.Logger.With(zap.String("after", "val"))

	assert.Equal(t, TestResults{
		LogScope:    `{"injected1":"val","injected2":"val"}`,
		LogFields:   `{"before":"val","injected1":"val","injected2":"val","after":"val","manual":"val"}`,
		SpanScope:   `{"after":"val","injected1":"val","injected2":"val"}`,
		MetricScope: `{"after":"val","injected1":"val","injected2":"val"}`,
	}, getScopes(t, tswa, logObs, spanObs, metricObs))

	// Drop one injected attribute
	tswa = telemetry.DropInjectedAttributes(tswa, "injected1")

	// Check scopes again
	assert.Equal(t, TestResults{
		LogScope:    `{"injected2":"val"}`,
		LogFields:   `{"before":"val","injected2":"val","after":"val","manual":"val"}`,
		SpanScope:   `{"after":"val","injected2":"val"}`,
		MetricScope: `{"after":"val","injected2":"val"}`,
	}, getScopes(t, tswa, logObs, spanObs, metricObs))
}

func TestTelemetryWithAttributes(t *testing.T) {
	t.Run("sdk", func(t *testing.T) {
		testTelemetryWithAttributes(t, true)
	})
	t.Run("generic", func(t *testing.T) {
		testTelemetryWithAttributes(t, false)
	})
}

func TestLoggerWithLevel(t *testing.T) {
	t.Run("raise_level_filters_messages", func(t *testing.T) {
		core, obs := observer.New(zap.DebugLevel)
		logger := zap.New(core)

		filtered := componentattribute.LoggerWithLevel(logger, zapcore.ErrorLevel)
		filtered.Debug("should be dropped")
		filtered.Info("should be dropped")
		filtered.Warn("should be dropped")
		filtered.Error("should appear")

		logs := obs.TakeAll()
		require.Len(t, logs, 1)
		assert.Equal(t, "should appear", logs[0].Message)
		assert.Equal(t, zapcore.ErrorLevel, logs[0].Level)
	})

	t.Run("same_level_preserves_messages", func(t *testing.T) {
		core, obs := observer.New(zap.InfoLevel)
		logger := zap.New(core)

		filtered := componentattribute.LoggerWithLevel(logger, zapcore.InfoLevel)
		filtered.Debug("should be dropped")
		filtered.Info("should appear")
		filtered.Warn("should appear")

		logs := obs.TakeAll()
		require.Len(t, logs, 2)
		assert.Equal(t, "should appear", logs[0].Message)
	})

	t.Run("with_preserves_level", func(t *testing.T) {
		core, obs := observer.New(zap.DebugLevel)
		logger := zap.New(core)

		filtered := componentattribute.LoggerWithLevel(logger, zapcore.WarnLevel)
		child := filtered.With(zap.String("key", "val"))
		child.Info("should be dropped")
		child.Warn("should appear")

		logs := obs.TakeAll()
		require.Len(t, logs, 1)
		assert.Equal(t, "should appear", logs[0].Message)
	})

	t.Run("lower_level_enables_debug", func(t *testing.T) {
		core, obs := observer.New(zap.InfoLevel)
		logger := zap.New(core)

		filtered := componentattribute.LoggerWithLevel(logger, zapcore.DebugLevel)
		filtered.Debug("debug should appear")
		filtered.Info("info should appear")

		logs := obs.TakeAll()
		require.Len(t, logs, 2)
		assert.Equal(t, "debug should appear", logs[0].Message)
		assert.Equal(t, zapcore.DebugLevel, logs[0].Level)
		assert.Equal(t, "info should appear", logs[1].Message)
	})
}
