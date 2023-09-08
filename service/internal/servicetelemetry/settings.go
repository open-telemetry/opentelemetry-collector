// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package servicetelemetry // import "go.opentelemetry.io/collector/internal/servicetelemetry"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/internal/status"
)

type Settings component.TelemetrySettingsBase[status.ServiceStatusFunc]

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
