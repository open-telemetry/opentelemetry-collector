// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

func TestNewTelemetry(t *testing.T) {
	tel := NewTelemetry()
	assert.NotNil(t, tel.Reader)
	assert.NotNil(t, tel.SpanRecorder)
	set := tel.NewTelemetrySettings()
	assert.Equal(t, configtelemetry.LevelDetailed, set.MetricsLevel) //nolint:staticcheck //SA1019
	assert.IsType(t, &sdktrace.TracerProvider{}, set.TracerProvider)
	assert.IsType(t, &sdkmetric.MeterProvider{}, set.MeterProvider)
	require.NoError(t, tel.Shutdown(context.Background()))
}
