// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/telemetry"
)

// Factories struct holds in a single type all component factories that
// can be handled by the Config.
type Factories struct {
	// Receivers maps receiver type names in the config to the respective factory.
	Receivers map[component.Type]receiver.Factory

	// Processors maps processor type names in the config to the respective factory.
	Processors map[component.Type]processor.Factory

	// Exporters maps exporter type names in the config to the respective factory.
	Exporters map[component.Type]exporter.Factory

	// Extensions maps extension type names in the config to the respective factory.
	Extensions map[component.Type]extension.Factory

	// Connectors maps connector type names in the config to the respective factory.
	Connectors map[component.Type]connector.Factory

	// Telemetry is the factory to create the telemetry providers for the service.
	Telemetry telemetry.Factory

	// ReceiverModules maps receiver types to their respective go modules.
	ReceiverModules map[component.Type]string

	// ProcessorModules maps processor types to their respective go modules.
	ProcessorModules map[component.Type]string

	// ExporterModules maps exporter types to their respective go modules.
	ExporterModules map[component.Type]string

	// ExtensionModules maps extension types to their respective go modules.
	ExtensionModules map[component.Type]string

	// ConnectorModules maps connector types to their respective go modules.
	ConnectorModules map[component.Type]string
}

// MakeFactoryMap takes a list of factories and returns a map with Factory type as keys.
// It returns a non-nil error when there are factories with duplicate type.
func MakeFactoryMap[T component.Factory](factories ...T) (map[component.Type]T, error) {
	fMap := map[component.Type]T{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate component factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
