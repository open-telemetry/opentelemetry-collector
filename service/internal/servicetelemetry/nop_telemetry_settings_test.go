// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package servicetelemetry

import (
	"testing"

	"github.com/stretchr/testify/require"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestNewNopSettings(t *testing.T) {
	set := NewNopTelemetrySettings()
	set.Status.Ready()
	require.NotNil(t, set)
	require.IsType(t, TelemetrySettings{}, set)
	require.Equal(t, zap.NewNop(), set.Logger)
	require.Equal(t, nooptrace.NewTracerProvider(), set.TracerProvider)
	require.Equal(t, noopmetric.NewMeterProvider(), set.MeterProvider)
	require.Equal(t, configtelemetry.LevelNone, set.MetricsLevel)
	require.Equal(t, pcommon.NewResource(), set.Resource)
	set.Status.ReportStatus(
		&component.InstanceID{},
		component.NewStatusEvent(component.StatusStarting),
	)
	set.Status.ReportOKIfStarting(&component.InstanceID{})

}
