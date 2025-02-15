// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/internal/telemetry/componentattribute"
	"go.opentelemetry.io/collector/pipeline"
)

type loggerCore interface {
	Without(fields ...string) zapcore.Core
}

func TestCore(t *testing.T) {
	core, observed := observer.New(zap.DebugLevel)
	logger := zap.New(core).With(zap.String("preexisting", "value"))

	attrs := attribute.NewSet(
		attribute.String(componentattribute.SignalKey, pipeline.SignalLogs.String()),
		attribute.String(componentattribute.ComponentIDKey, "filelog"),
	)

	parent := componentattribute.NewLogger(logger, &attrs)
	parent.Info("test parent before child")
	childCore := parent.Core().(loggerCore).Without(string(componentattribute.SignalKey))
	child := zap.New(childCore)
	child.Info("test child")
	parent.Info("test parent after child")

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
