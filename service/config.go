// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/collector/service"

import (
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/pipelines"
	"go.opentelemetry.io/collector/service/telemetry"
)

// Config defines the configurable components of the Service.
type Config struct {
	// Telemetry is the configuration for collector's own telemetry.
	Telemetry telemetry.Config `mapstructure:"telemetry"`

	// Extensions are the ordered list of extensions configured for the service.
	Extensions extensions.Config `mapstructure:"extensions,omitempty"`

	// Pipelines are the set of data pipelines configured for the service.
	Pipelines pipelines.Config `mapstructure:"pipelines"`
}
