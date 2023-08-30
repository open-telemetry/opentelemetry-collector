// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package servicetelemetry // import "go.opentelemetry.io/collector/internal/servicetelemetry"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/internal/status"
)

type Settings component.TelemetrySettingsBase[status.ReportStatusFunc]

func (s Settings) ToComponentTelemetrySettings(instanceID *component.InstanceID) component.TelemetrySettings {
	notifier := status.NewNotifier(instanceID, s.ReportComponentStatus)
	return component.TelemetrySettings{
		Logger:         s.Logger,
		TracerProvider: s.TracerProvider,
		MeterProvider:  s.MeterProvider,
		MetricsLevel:   s.MetricsLevel,
		Resource:       s.Resource,
		ReportComponentStatus: func(status component.Status, options ...component.StatusEventOption) {
			notifier.Event(status, options...)
		},
	}
}
