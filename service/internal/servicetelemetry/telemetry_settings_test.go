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
		Status: status.NewReporter(
			func(*component.InstanceID, *component.StatusEvent) {},
			func(err error) { require.NoError(t, err) }),
	}
	set.Status.Ready()
	set.Status.ReportStatus(
		&component.InstanceID{},
		component.NewStatusEvent(component.StatusStarting),
	)
	set.Status.ReportOKIfStarting(&component.InstanceID{})

	compSet := set.ToComponentTelemetrySettings(&component.InstanceID{})
	compSet.ReportStatus(component.NewStatusEvent(component.StatusStarting))
}
