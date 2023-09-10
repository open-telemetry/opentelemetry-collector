// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package servicetelemetry // import "go.opentelemetry.io/collector/service/internal/servicetelemetry"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/internal/status"
)

// Settings mirrors component.TelemetrySettings except for the method signature of
// ReportComponentStatus. The service level Settings is not bound a specific component, and
// therefore takes a component.InstanceID as an argument.
type Settings component.TelemetrySettingsBase[status.ServiceStatusFunc]

// ToComponentTelemetrySettings returns a TelemetrySettings for a specific component derived from
// this service level Settings object.
func (s Settings) ToComponentTelemetrySettings(id *component.InstanceID) component.TelemetrySettings {
	return component.TelemetrySettings{
		Logger:                s.Logger,
		TracerProvider:        s.TracerProvider,
		MeterProvider:         s.MeterProvider,
		MetricsLevel:          s.MetricsLevel,
		Resource:              s.Resource,
		ReportComponentStatus: status.NewComponentStatusFunc(id, s.ReportComponentStatus),
	}
}
