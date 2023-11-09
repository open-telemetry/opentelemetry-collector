// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package servicetelemetry // import "go.opentelemetry.io/collector/service/internal/servicetelemetry"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/internal/status"
)

// TelemetrySettings mirrors component.TelemetrySettings except for the mechanism for reporting
// status. Service-level status reporting has additional methods which can report status for
// components by their InstanceID whereas the component versions are tied to a specific component.
type TelemetrySettings struct {
	*component.TelemetrySettingsBase
	Status *status.Reporter
}

// ToComponentTelemetrySettings returns a TelemetrySettings for a specific component derived from
// this service level Settings object.
func (s TelemetrySettings) ToComponentTelemetrySettings(id *component.InstanceID) component.TelemetrySettings {
	return component.TelemetrySettings{
		TelemetrySettingsBase: &component.TelemetrySettingsBase{
			Logger:         s.Logger,
			TracerProvider: s.TracerProvider,
			MeterProvider:  s.MeterProvider,
			MetricsLevel:   s.MetricsLevel,
			Resource:       s.Resource,
		},

		ReportComponentStatus: status.NewComponentStatusFunc(id, s.Status.ReportComponentStatus),
	}
}
