// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package wrappers // import "go.opentelemetry.io/collector/service/internal/status/wrappers"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
)

type componentWrapper struct {
	component component.Component
	telemetry *component.TelemetrySettings
}

func (cw *componentWrapper) Start(ctx context.Context, host component.Host) error {
	_ = cw.telemetry.ReportComponentStatus(component.NewStatusEvent(component.StatusStarting))
	err := cw.component.Start(ctx, host)
	_ = cw.telemetry.ReportComponentStatus(
		component.NewEventFromError(err, component.StatusPermanentError),
	)
	if err != nil && !componenterror.IsRecoverable(err) {
		return err
	}
	return nil
}

func (cw *componentWrapper) Shutdown(ctx context.Context) error {
	_ = cw.telemetry.ReportComponentStatus(component.NewStatusEvent(component.StatusStopping))
	if err := cw.component.Shutdown(ctx); err != nil {
		_ = cw.telemetry.ReportComponentStatus(
			component.NewEventFromError(err, component.StatusPermanentError),
		)
		return err
	}
	_ = cw.telemetry.ReportComponentStatus(component.NewStatusEvent(component.StatusStopped))
	return nil
}

func WrapComponent(c component.Component, telemetry *component.TelemetrySettings) component.Component {
	return &componentWrapper{
		component: c,
		telemetry: telemetry,
	}
}
