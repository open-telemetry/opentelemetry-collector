// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package hostcapabilities provides interfaces that can be implemented by the host
// to provide additional capabilities.
package hostcapabilities // import "go.opentelemetry.io/collector/service/hostcapabilities"

import (
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/internal/moduleinfo"
)

// ModuleInfo is an interface that may be implemented by the host to provide
// information about modules that were used to build the host.
type ModuleInfo interface {
	// GetModuleInfos returns the module information for the host
	// i.e. Receivers, Processors, Exporters, Extensions, and Connectors
	GetModuleInfos() moduleinfo.ModuleInfos
}

// ExposeExporters is an interface that may be implemented by the host to provide
// access to the exporters that were used to build the host.
// Deprecated: [v0.121.0] Will be removed in Service 1.0.
// See: https://github.com/open-telemetry/opentelemetry-collector/issues/7370 for service 1.0
type ExposeExporters interface {
	GetExporters() map[pipeline.Signal]map[component.ID]component.Component
}

// ComponentFactory is an interface that may be implemented by the host to
// provide a component's factory
type ComponentFactory interface {
	// GetFactory returns the component factory for the given
	// component type
	GetFactory(kind component.Kind, componentType component.Type) component.Factory
}

// ZPages is an interface that may be implemented by the host to provide
// a zPages HTTP mux, for components to provide or expose diagnostics.
type ZPages interface {
	// GetZPagesMux returns the zPages HTTP mux.
	//
	// The returned mux can be expected to handle the following routes:
	//
	//   GET /debug/extensionz
	//   GET /debug/featurez
	//   GET /debug/pipelinez
	//   GET /debug/servicez
	//
	// If supported by the service's tracer provider, it may also handle:
	//
	//   GET /debug/tracez
	//
	// Additional routes may be added to the mux by extensions.
	GetZPagesMux() *http.ServeMux
}
