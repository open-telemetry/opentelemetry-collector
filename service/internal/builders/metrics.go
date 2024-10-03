// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builders // import "go.opentelemetry.io/collector/service/internal/builders"

import (
	"context"
	"reflect"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type metricsComponent interface {
	component.Component
	consumer.Metrics
}

type debugMetrics struct {
	id  component.ID
	set component.TelemetrySettings
	mc  metricsComponent
}

func (dc *debugMetrics) Start(ctx context.Context, host component.Host) error {
	return dc.mc.Start(ctx, host)
}

func (dc *debugMetrics) Shutdown(ctx context.Context) error {
	return dc.mc.Shutdown(ctx)
}

func (dc *debugMetrics) Capabilities() consumer.Capabilities {
	return dc.mc.Capabilities()
}

func (dc *debugMetrics) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	err := dc.mc.ConsumeMetrics(ctx, md)
	if err == nil {
		return nil
	}
	val := reflect.ValueOf(err)
	if val.Kind() == reflect.Ptr && val.IsNil() {
		dc.set.Logger.Error("nil error that is not nil",
			zap.String("error_type", val.Type().String()),
			zap.Int("num_data_points", md.DataPointCount()))
		return nil
	}
	return err
}

func wrapMetrics(id component.ID, set component.TelemetrySettings, mc metricsComponent) metricsComponent {
	return &debugMetrics{id: id, set: set, mc: mc}
}
