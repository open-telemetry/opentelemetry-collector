// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log/logtest"
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
				core = NewOTelTeeCoreWithAttributes(core, recorder, "testinstr", zap.DebugLevel, attribute.NewSet())
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
			withAttributes: func(inner zapcore.Core, sampler zapcore.Core, attrs attribute.Set) zapcore.Core {
				return zapcore.NewSamplerWithOptions(tryWithAttributeSet(inner, attrs), tick, first, thereafter)
			},
			expectedAttrs: []string{"foo", "bar", "foo", "bar"},
		},
		{
			name: "cloned-sampler",
			withAttributes: func(inner zapcore.Core, sampler zapcore.Core, attrs attribute.Set) zapcore.Core {
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
