// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log/logtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/internal/telemetry/componentattribute"
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
	core = componentattribute.NewConsoleCoreWithAttributes(core, attribute.NewSet())
	return core, observed
}

func checkZapLogs(t *testing.T, observed *observer.ObservedLogs) {
	observedLogs := observed.All()
	require.Len(t, observedLogs, 3)

	parentContext := map[string]string{
		"preexisting":                     "value",
		componentattribute.SignalKey:      pipeline.SignalLogs.String(),
		componentattribute.ComponentIDKey: "filelog",
	}
	childContext := map[string]string{
		"preexisting":                     "value",
		componentattribute.ComponentIDKey: "filelog",
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
		attribute.String(componentattribute.SignalKey, pipeline.SignalLogs.String()),
		attribute.String(componentattribute.ComponentIDKey, "filelog"),
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
				core = componentattribute.NewOTelTeeCoreWithAttributes(core, recorder, "testinstr", zap.DebugLevel, attribute.NewSet())
				return zap.New(core), logRecorder{zapLogs: observed, otelLogs: recorder}
			},
			check: func(t *testing.T, rec logRecorder) {
				checkZapLogs(t, rec.zapLogs)

				recorder := rec.otelLogs

				logAttributes := make(map[string]attribute.Set)
				for _, scope := range recorder.Result() {
					require.Equal(t, "testinstr", scope.Name)
					for _, record := range scope.Records {
						logAttributes[record.Body().String()] = scope.Attributes
					}
				}

				childAttrs := attribute.NewSet(
					attribute.String(componentattribute.ComponentIDKey, "filelog"),
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

			parent := componentattribute.ZapLoggerWithAttributes(logger, attrs)
			parent.Info("test parent before child")
			child := componentattribute.ZapLoggerWithAttributes(parent, componentattribute.RemoveAttributes(attrs, componentattribute.SignalKey))
			child.Info("test child")
			parent.Info("test parent after child")

			test.check(t, state)
		})
	}
}
