// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package requestmiddleware // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/requestmiddleware"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
)

// RequestMiddlewareSettings provides context about the exporter request pipeline
// to middleware extensions. This type is re-exported as
// xexporterhelper.RequestMiddlewareSettings for use by extension authors.
type RequestMiddlewareSettings struct {
	Signal    pipeline.Signal
	ID        component.ID
	Telemetry component.TelemetrySettings
}
