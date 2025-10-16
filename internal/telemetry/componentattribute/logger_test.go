// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/logtest"
	"go.opentelemetry.io/otel/log/noop"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/pipeline"
)

type logRecorder struct {
	zapLogs  *observer.ObservedLogs
	otelLogs *logtest.Recorder
}
type test struct {
	name         string
	createLogger func() (*zap.Logger, logRecorder)
	check        func(*testing.T, logRecorder)
}

func createZapCore() (zapcore.Core, *observer.ObservedLogs) {
	core, observed := observer.New(zap.DebugLevel)
	core = core.With([]zapcore.Field{zap.String("preexisting", "value")})
	core = NewConsoleCoreWithAttributes(core, attribute.NewSet())
	return core, observed
}

func checkZapLogs(t *testing.T, observed *observer.ObservedLogs) {
	observedLogs := observed.All()
	require.Len(t, observedLogs, 3)

	parentContext := map[string]string{
		"preexisting":  "value",
		SignalKey:      pipeline.SignalLogs.String(),
		ComponentIDKey: "filelog",
	}
	childContext := map[string]string{
		"preexisting":  "value",
		ComponentIDKey: "filelog",
	}

	require.Equal(t, "test parent before child", observedLogs[0].Message)
	require.Len(t, observedLogs[0].Context, len(parentContext))
	for _, field := range observedLogs[0].Context {
		require.Equal(t, parentContext[field.Key], field.String)
	}

	require.Equal(t, "test child", observedLogs[1].Message)
	require.Len(t, observedLogs[1].Context, len(childContext))
	for _, field := range observedLogs[1].Context {
		require.Equal(t, childContext[field.Key], field.String)
	}

	require.Equal(t, "test parent after child", observedLogs[2].Message)
	require.Len(t, observedLogs[2].Context, len(parentContext))
	for _, field := range observedLogs[2].Context {
		require.Equal(t, parentContext[field.Key], field.String)
	}
}

func TestCore(t *testing.T) {
	attrs := attribute.NewSet(
		attribute.String(SignalKey, pipeline.SignalLogs.String()),
		attribute.String(ComponentIDKey, "filelog"),
	)

	tests := []test{
		{
			name: "console",
			createLogger: func() (*zap.Logger, logRecorder) {
				core, observed := createZapCore()
				return zap.New(core), logRecorder{zapLogs: observed}
			},
			check: func(t *testing.T, rec logRecorder) {
				checkZapLogs(t, rec.zapLogs)
			},
		},
		{
			name: "console + otel",
			createLogger: func() (*zap.Logger, logRecorder) {
				core, observed := createZapCore()
				recorder := logtest.NewRecorder()
				core = NewOTelTeeCoreWithAttributes(core, recorder, "testinstr", attribute.NewSet())
				return zap.New(core), logRecorder{zapLogs: observed, otelLogs: recorder}
			},
			check: func(t *testing.T, rec logRecorder) {
				checkZapLogs(t, rec.zapLogs)

				recorder := rec.otelLogs

				logAttributes := make(map[string]attribute.Set)
				for scope, records := range recorder.Result() {
					require.Equal(t, "testinstr", scope.Name)
					for _, record := range records {
						logAttributes[record.Body.String()] = scope.Attributes
					}
				}

				childAttrs := attribute.NewSet(
					attribute.String(ComponentIDKey, "filelog"),
				)

				assert.Equal(t, map[string]attribute.Set{
					"test parent before child": attrs,
					"test child":               childAttrs,
					"test parent after child":  attrs,
				}, logAttributes)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger, state := test.createLogger()

			parent := ZapLoggerWithAttributes(logger, attrs)
			parent.Info("test parent before child")
			child := ZapLoggerWithAttributes(parent, RemoveAttributes(attrs, SignalKey))
			child.Info("test child")
			parent.Info("test parent after child")

			test.check(t, state)
		})
	}
}

func TestNewOTelTeeCoreWithAttributes(t *testing.T) {
	t.Run("copy_accepted_logs", func(t *testing.T) {
		// Only log at Info level. Debug level logs should not be copied to the LoggerProvider.
		observerCore, _ := observer.New(zap.InfoLevel)
		recorder := logtest.NewRecorder()
		core := NewOTelTeeCoreWithAttributes(observerCore, recorder, "scope", attribute.NewSet())
		timestamp := time.Now()
		logger := zap.New(core, zap.WithClock(constantClock(timestamp)))

		loggerWith := logger.With(zap.String("logger_key", "logger_value"))
		loggerWith.Info("message", zap.String("record_key", "record_value"))
		loggerWith.Debug("dropped") // should not be recorded due to observer's level

		logtest.AssertEqual(t, logtest.Recording{
			logtest.Scope{Name: "scope"}: []logtest.Record{{
				Context:      context.Background(),
				Timestamp:    timestamp,
				Severity:     log.SeverityInfo,
				SeverityText: "info",
				Body:         log.StringValue("message"),
				Attributes: []log.KeyValue{
					log.String("logger_key", "logger_value"),
					log.String("record_key", "record_value"),
				},
			}},
		}, recorder.Result())

		require.NoError(t, logger.Sync()) // no-op for otelzap
	})
	t.Run("nop_core", func(t *testing.T) {
		// Using zapcore.NewNopCore should result in no logs being sent to the LoggerProvider.
		recorder := logtest.NewRecorder()
		core := NewOTelTeeCoreWithAttributes(zapcore.NewNopCore(), recorder, "scope", attribute.NewSet())
		logger := zap.New(core)

		logger.Error("message")
		logtest.AssertEqual(t, logtest.Recording{logtest.Scope{Name: "scope"}: nil}, recorder.Result())
	})
	t.Run("noop_loggerprovider", func(t *testing.T) {
		// Using a noop LoggerProvider should not impact the main zap core.
		observerCore, observedLogs := observer.New(zap.InfoLevel)
		noopProvider := noop.NewLoggerProvider()
		core := NewOTelTeeCoreWithAttributes(observerCore, noopProvider, "scope", attribute.NewSet())
		logger := zap.New(core)

		logger.Info("message", zap.String("key", "value"))
		logger.Debug("dropped") // should not be recorded due to observer's level

		assert.Equal(t, 1, observedLogs.Len())
	})
	t.Run("direct_write", func(t *testing.T) {
		observerCore, observedLogs := observer.New(zap.InfoLevel)
		recorder := logtest.NewRecorder()
		core := NewOTelTeeCoreWithAttributes(observerCore, recorder, "scope", attribute.NewSet())

		// Per https://pkg.go.dev/go.uber.org/zap/zapcore#Core:
		//
		//   If called, Write should always log the Entry and Fields;
		//   it should not replicate the logic of Check.
		//
		// Even though the observer has been configured with Info level,
		// Debug level logs should therefore be written.
		require.NoError(t, core.Write(zapcore.Entry{
			Level:   zapcore.DebugLevel,
			Message: "m",
		}, []zapcore.Field{{
			Key:    "k",
			Type:   zapcore.StringType,
			String: "s",
		}}))

		logtest.AssertEqual(t, logtest.Recording{
			logtest.Scope{Name: "scope"}: []logtest.Record{{
				Context:      context.Background(),
				Severity:     log.SeverityDebug,
				SeverityText: "debug",
				Body:         log.StringValue("m"),
				Attributes:   []log.KeyValue{log.String("k", "s")},
			}},
		}, recorder.Result())

		assert.Equal(t, 1, observedLogs.Len())
	})
}

type constantClock time.Time

func (c constantClock) Now() time.Time                       { return time.Time(c) }
func (c constantClock) NewTicker(time.Duration) *time.Ticker { return &time.Ticker{} }

func TestToZapFields(t *testing.T) {
	tests := []struct {
		attrs    attribute.Set
		expected []zap.Field
	}{
		{
			attrs: attribute.NewSet(
				attribute.String("string_key", "string_value"),
			),
			expected: []zap.Field{
				zap.String("string_key", "string_value"),
			},
		},
		{
			attrs: attribute.NewSet(
				attribute.Bool("bool_key", true),
			),
			expected: []zap.Field{
				zap.Bool("bool_key", true),
			},
		},
		{
			attrs: attribute.NewSet(
				attribute.Int64("int64_key", 42),
			),
			expected: []zap.Field{
				zap.Int64("int64_key", 42),
			},
		},
		{
			attrs: attribute.NewSet(
				attribute.Float64("float64_key", 3.14),
			),
			expected: []zap.Field{
				zap.Float64("float64_key", 3.14),
			},
		},
		{
			attrs: attribute.NewSet(
				attribute.BoolSlice("bool_slice_key", []bool{true, false, true}),
			),
			expected: []zap.Field{
				zap.Bools("bool_slice_key", []bool{true, false, true}),
			},
		},
		{
			attrs: attribute.NewSet(
				attribute.Int64Slice("int64_slice_key", []int64{1, 2, 3}),
			),
			expected: []zap.Field{
				zap.Int64s("int64_slice_key", []int64{1, 2, 3}),
			},
		},
		{
			attrs: attribute.NewSet(
				attribute.Float64Slice("float64_slice_key", []float64{1.1, 2.2, 3.3}),
			),
			expected: []zap.Field{
				zap.Float64s("float64_slice_key", []float64{1.1, 2.2, 3.3}),
			},
		},
		{
			attrs: attribute.NewSet(
				attribute.StringSlice("string_slice_key", []string{"a", "b", "c"}),
			),
			expected: []zap.Field{
				zap.Strings("string_slice_key", []string{"a", "b", "c"}),
			},
		},
	}

	for _, tt := range tests {
		name := "<empty>"
		if tt.attrs.Len() > 0 {
			attr, ok := tt.attrs.Get(0)
			if ok {
				name = string(attr.Key)
			}
		}
		t.Run(name, func(t *testing.T) {
			result := ToZapFields(tt.attrs)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestLoggerWith(t *testing.T) {
	core, observed := createZapCore()
	logger := zap.New(core)
	logger = logger.With(zap.String("postexisting", "value"))
	logger = ZapLoggerWithAttributes(logger, attribute.NewSet(attribute.String("component.attr", "value")))
	logger.Info("test")

	observedLogs := observed.All()
	require.Len(t, observedLogs, 1)
	expectedContext := []string{
		"preexisting",
		"component.attr",
		"postexisting",
	}
	require.Equal(t, "test", observedLogs[0].Message)
	require.Len(t, observedLogs[0].Context, len(expectedContext))
	for i, field := range observedLogs[0].Context {
		require.Equal(t, expectedContext[i], field.Key)
	}
}
