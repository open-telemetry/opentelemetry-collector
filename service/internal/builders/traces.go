// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builders // import "go.opentelemetry.io/collector/service/internal/builders"

import (
	"context"
	"reflect"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type tracesComponent interface {
	component.Component
	consumer.Traces
}

type debugTraces struct {
	id  component.ID
	set component.TelemetrySettings
	tc  tracesComponent
}

func (dc *debugTraces) Start(ctx context.Context, host component.Host) error {
	return dc.tc.Start(ctx, host)
}

func (dc *debugTraces) Shutdown(ctx context.Context) error {
	return dc.tc.Shutdown(ctx)
}

func (dc *debugTraces) Capabilities() consumer.Capabilities {
	return dc.tc.Capabilities()
}

func (dc *debugTraces) ConsumeTraces(ctx context.Context, tr ptrace.Traces) error {
	err := dc.tc.ConsumeTraces(ctx, tr)
	if err == nil {
		return nil
	}
	val := reflect.ValueOf(err)
	if val.Kind() == reflect.Ptr && val.IsNil() {
		dc.set.Logger.Error("nil error that is not nil",
			zap.String("error_type", val.Type().String()))
		return nil
	}
	return err
}

func wrapTraces(id component.ID, set component.TelemetrySettings, tc tracesComponent) tracesComponent {
	return &debugTraces{id: id, set: set, tc: tc}
}
