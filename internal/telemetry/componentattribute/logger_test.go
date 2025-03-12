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

	"go.opentelemetry.io/collector/internal/telemetry/componentattribute"
	"go.opentelemetry.io/collector/pipeline"
)

func TestCore(t *testing.T) {
	lp := logtest.NewRecorder()
	logger := zap.New(componentattribute.NewServiceZapCore(lp, "testinstr", nil, attribute.NewSet()))

	attrs := attribute.NewSet(
		attribute.String(componentattribute.SignalKey, pipeline.SignalLogs.String()),
		attribute.String(componentattribute.ComponentIDKey, "filelog"),
	)

	parent := componentattribute.ZapLoggerWithAttributes(logger, attrs)
	parent.Info("test parent before child")
	child := componentattribute.ZapLoggerWithAttributes(parent, componentattribute.RemoveAttributes(attrs, componentattribute.SignalKey))
	child.Info("test child")
	parent.Info("test parent after child")

	logAttributes := make(map[string]attribute.Set)
	for _, scope := range lp.Result() {
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
}
