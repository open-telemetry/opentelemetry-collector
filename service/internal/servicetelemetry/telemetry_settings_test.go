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
	"go.opentelemetry.io/collector/service/internal/status"
)

func TestSettings(t *testing.T) {
	set := TelemetrySettings{
		Logger:         zap.NewNop(),
		TracerProvider: nooptrace.NewTracerProvider(),
		MeterProvider:  noopmetric.NewMeterProvider(),
		MetricsLevel:   configtelemetry.LevelNone,
		Resource:       pcommon.NewResource(),
		Status:         status.NewReporter(func(*component.InstanceID, *component.StatusEvent) {}),
	}
	set.Status.Ready()
	require.NoError(t,
		set.Status.ReportComponentStatus(
			&component.InstanceID{},
			component.NewStatusEvent(component.StatusStarting),
		),
	)
	require.NoError(t, set.Status.ReportComponentOKIfStarting(&component.InstanceID{}))

	compSet := set.ToComponentTelemetrySettings(&component.InstanceID{})
	require.NoError(t, compSet.ReportComponentStatus(component.NewStatusEvent(component.StatusStarting)))
}
