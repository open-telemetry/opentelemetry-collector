// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package servicetelemetry

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestNewNopSettings(t *testing.T) {
	set := NewNopSettings()

	require.NotNil(t, set)
	require.IsType(t, Settings{}, set)
	require.Equal(t, zap.NewNop(), set.Logger)
	require.Equal(t, trace.NewNoopTracerProvider(), set.TracerProvider)
	require.Equal(t, noop.NewMeterProvider(), set.MeterProvider)
	require.Equal(t, configtelemetry.LevelNone, set.MetricsLevel)
	require.Equal(t, pcommon.NewResource(), set.Resource)
	require.NoError(t, set.ReportComponentStatus(&component.InstanceID{}, component.StatusStarting))
}
