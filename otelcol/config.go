// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service"
)

var (
	errMissingExporters       = errors.New("no exporter configuration specified in config")
	errMissingReceivers       = errors.New("no receiver configuration specified in config")
	errEmptyConfigurationFile = errors.New("empty configuration file")
)

// Config defines the configuration for the various elements of collector or agent.
type Config struct {
	// Receivers is a map of ComponentID to Receivers.
	Receivers map[component.ID]component.Config `mapstructure:"receivers"`

	// Exporters is a map of ComponentID to Exporters.
	Exporters map[component.ID]component.Config `mapstructure:"exporters"`

	// Processors is a map of ComponentID to Processors.
	Processors map[component.ID]component.Config `mapstructure:"processors"`

	// Connectors is a map of ComponentID to connectors.
	Connectors map[component.ID]component.Config `mapstructure:"connectors"`

	// Extensions is a map of ComponentID to extensions.
	Extensions map[component.ID]component.Config `mapstructure:"extensions"`

	Service service.Config `mapstructure:"service"`
}
