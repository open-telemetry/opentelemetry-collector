// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component // import "go.opentelemetry.io/collector/component"

// Host represents the entity that is hosting a Component. It is used to allow communication
// between the Component and its host (normally the service.Collector is the host).
type Host interface {
	// ReportFatalError is used to report to the host that the component
	// encountered a fatal error (i.e.: an error that the instance can't recover
	// from) after its start function had already returned.
	//
	// ReportFatalError should be called by the component anytime after Component.Start() ends and
	// before Component.Shutdown() begins.
	// Deprecated: [0.87.0] Use TelemetrySettings.ReportComponentStatus instead (with an event
	// component.StatusFatalError)
	ReportFatalError(err error)

	// GetFactory of the specified kind. Returns the factory for a component type.
	// This allows components to create other components. For example:
	//   func (r MyReceiver) Start(host component.Host) error {
	//     apacheFactory := host.GetFactory(KindReceiver,"apache").(receiver.Factory)
	//     receiver, err := apacheFactory.CreateMetrics(...)
	//     ...
	//   }
	//
	// GetFactory can be called by the component anytime after Component.Start() begins and
	// until Component.Shutdown() ends. Note that the component is responsible for destroying
	// other components that it creates.
	GetFactory(kind Kind, componentType Type) Factory

	// GetExtensions returns the map of extensions. Only enabled and created extensions will be returned.
	// Typically is used to find an extension by type or by full config name. Both cases
	// can be done by iterating the returned map. There are typically very few extensions,
	// so there are no performance implications due to iteration.
	//
	// GetExtensions can be called by the component anytime after Component.Start() begins and
	// until Component.Shutdown() ends.
	GetExtensions() map[ID]Component

	// GetExporters returns the map of exporters. Only enabled and created exporters will be returned.
	// Typically is used to find exporters by type or by full config name. Both cases
	// can be done by iterating the returned map. There are typically very few exporters,
	// so there are no performance implications due to iteration.
	// This returns a map by DataType of maps by exporter configs to the exporter instance.
	// Note that an exporter with the same name may be attached to multiple pipelines and
	// thus we may have an instance of the exporter for multiple data types.
	// This is an experimental function that may change or even be removed completely.
	//
	// GetExporters can be called by the component anytime after Component.Start() begins and
	// until Component.Shutdown() ends.
	//
	// Deprecated: [0.79.0] This function will be removed in the future.
	// Several components in the contrib repository use this function so it cannot be removed
	// before those cases are removed. In most cases, use of this function can be replaced by a
	// connector. See https://github.com/open-telemetry/opentelemetry-collector/issues/7370 and
	// https://github.com/open-telemetry/opentelemetry-collector/pull/7390#issuecomment-1483710184
	// for additional information.
	GetExporters() map[DataType]map[ID]Component
}
