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
	// Only log at Info level. Debug level logs should not be copied to the LoggerProvider.
	t.Run("copy_accepted_logs", func(t *testing.T) {
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

func TestSamplerCore(t *testing.T) {
	tick := time.Second
	// Drop identical messages after the first two
	first := 2
	thereafter := 0

	type testCase struct {
		name           string
		withAttributes func(inner zapcore.Core, sampler zapcore.Core, attrs attribute.Set) zapcore.Core
		expectedAttrs  []string
	}
	testCases := []testCase{
		{
			name: "new-sampler",
			withAttributes: func(inner zapcore.Core, _ zapcore.Core, attrs attribute.Set) zapcore.Core {
				return zapcore.NewSamplerWithOptions(tryWithAttributeSet(inner, attrs), tick, first, thereafter)
			},
			expectedAttrs: []string{"foo", "bar", "foo", "bar"},
		},
		{
			name: "cloned-sampler",
			withAttributes: func(_ zapcore.Core, sampler zapcore.Core, attrs attribute.Set) zapcore.Core {
				return tryWithAttributeSet(sampler, attrs)
			},
			expectedAttrs: []string{"foo", "bar"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inner, obs := observer.New(zapcore.DebugLevel)
			inner = NewConsoleCoreWithAttributes(inner, attribute.NewSet(attribute.String("test", "foo")))

			sampler1 := NewSamplerCoreWithAttributes(inner, tick, first, thereafter)
			loggerFoo := zap.New(sampler1)

			sampler2 := tc.withAttributes(inner, sampler1, attribute.NewSet(attribute.String("test", "bar")))
			loggerBar := zap.New(sampler2)

			// If the two samplers share their counters, only the first two messages will go through.
			// If they are independent, the first three and the fifth will go through.
			loggerFoo.Info("test")
			loggerBar.Info("test")
			loggerFoo.Info("test")
			loggerFoo.Info("test")
			loggerBar.Info("test")
			loggerBar.Info("test")

			var attrs []string
			for _, log := range obs.All() {
				var fooValue string
				for _, field := range log.Context {
					if field.Key == "test" {
						fooValue = field.String
					}
				}
				attrs = append(attrs, fooValue)
			}
			assert.Equal(t, tc.expectedAttrs, attrs)
		})
	}
}

// Worst case scenario for the reflect spell in samplerCoreWithAttributes
type crazySampler struct {
	Core int
}

var _ zapcore.Core = (*crazySampler)(nil)

func (s *crazySampler) Enabled(zapcore.Level) bool {
	return true
}

func (s *crazySampler) With([]zapcore.Field) zapcore.Core {
	return s
}

func (s *crazySampler) Check(_ zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return ce
}

func (s *crazySampler) Write(zapcore.Entry, []zapcore.Field) error {
	return nil
}

func (s *crazySampler) Sync() error {
	return nil
}

func TestSamplerCorePanic(t *testing.T) {
	sampler := NewSamplerCoreWithAttributes(zapcore.NewNopCore(), 1, 1, 1)
	sampler.(*samplerCoreWithAttributes).Core = &crazySampler{}
	assert.PanicsWithValue(t, "Unexpected Zap sampler type; see github.com/open-telemetry/opentelemetry-collector/issues/13014", func() {
		tryWithAttributeSet(sampler, attribute.NewSet())
	})
}

type constantClock time.Time

func (c constantClock) Now() time.Time                       { return time.Time(c) }
func (c constantClock) NewTicker(time.Duration) *time.Ticker { return &time.Ticker{} }
