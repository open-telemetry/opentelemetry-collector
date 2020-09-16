// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"context"

	"github.com/spf13/viper"

	"go.opentelemetry.io/collector/config/configmodels"
)

// Component is either a receiver, exporter, processor or extension.
type Component interface {
	// Start tells the component to start. Host parameter can be used for communicating
	// with the host after Start() has already returned. If error is returned by
	// Start() then the collector startup will be aborted.
	// If this is an exporter component it may prepare for exporting
	// by connecting to the endpoint.
	//
	// If the component needs to perform a long-running starting operation then it is recommended
	// that Start() returns quickly and the long-running operation is performed in background.
	// In that case make sure that the long-running operation does not use the context passed
	// to Start() function since that context will be cancelled soon and can abort the long-running operation.
	// Create a new context from the context.Background() for long-running operations.
	Start(ctx context.Context, host Host) error

	// Shutdown is invoked during service shutdown.
	//
	// If there are any background operations running by the component they must be aborted as soon as possible.
	// Remember that if you started any long-running background operation from the Start() method that operation
	// must be also cancelled.
	Shutdown(ctx context.Context) error
}

// Kind specified one of the 4 components kinds, see consts below.
type Kind int

const (
	_ Kind = iota // skip 0, start types from 1.
	KindReceiver
	KindProcessor
	KindExporter
	KindExtension
)

// Host represents the entity that is hosting a Component. It is used to allow communication
// between the Component and its host (normally the service.Application is the host).
type Host interface {
	// ReportFatalError is used to report to the host that the extension
	// encountered a fatal error (i.e.: an error that the instance can't recover
	// from) after its start function had already returned.
	ReportFatalError(err error)

	// GetFactory of the specified kind. Returns the factory for a component type.
	// This allows components to create other components. For example:
	//   func (r MyReceiver) Start(host component.Host) error {
	//     apacheFactory := host.GetFactory(KindReceiver,"apache").(component.ReceiverFactory)
	//     receiver, err := apacheFactory.CreateMetricsReceiver(...)
	//     ...
	//   }
	// GetFactory can be called by the component anytime after Start() begins and
	// until Shutdown() is called. Note that the component is responsible for destroying
	// other components that it creates.
	GetFactory(kind Kind, componentType configmodels.Type) Factory

	// Return map of extensions. Only enabled and created extensions will be returned.
	// Typically is used to find an extension by type or by full config name. Both cases
	// can be done by iterating the returned map. There are typically very few extensions
	// so there there is no performance implications due to iteration.
	GetExtensions() map[configmodels.Extension]ServiceExtension

	// Return map of exporters. Only enabled and created exporters will be returned.
	// Typically is used to find exporters by type or by full config name. Both cases
	// can be done by iterating the returned map. There are typically very few exporters
	// so there there is no performance implications due to iteration.
	// This returns a map by DataType of maps by exporter configs to the exporter instance.
	// Note that an exporter with the same name may be attached to multiple pipelines and
	// thus we may have an instance of the exporter for multiple data types.
	// This is an experimental function that may change or even be removed completely.
	GetExporters() map[configmodels.DataType]map[configmodels.Exporter]Exporter
}

// Factory interface must be implemented by all component factories.
type Factory interface {
	// Type gets the type of the component created by this factory.
	Type() configmodels.Type
}

// ConfigUnmarshaler interface is an optional interface that if implemented by a Factory,
// the configuration loading system will use to unmarshal the config.
type ConfigUnmarshaler interface {
	// Unmarshal is a function that un-marshals a viper data into a config struct in a custom way.
	// componentViperSection *viper.Viper
	//   The config for this specific component. May be nil or empty if no config available.
	// intoCfg interface{}
	//   An empty interface wrapping a pointer to the config struct to unmarshal into.
	Unmarshal(componentViperSection *viper.Viper, intoCfg interface{}) error
}

// CustomUnmarshaler is a function that un-marshals a viper data into a config struct
// in a custom way.
// componentViperSection *viper.Viper
//   The config for this specific component. May be nil or empty if no config available.
// intoCfg interface{}
//   An empty interface wrapping a pointer to the config struct to unmarshal into.
type CustomUnmarshaler func(componentViperSection *viper.Viper, intoCfg interface{}) error

// ApplicationStartInfo is the information that is logged at the application start and
// passed into each component. This information can be overridden in custom builds.
type ApplicationStartInfo struct {
	// Executable file name, e.g. "otelcol".
	ExeName string

	// Long name, used e.g. in the logs.
	LongName string

	// Version string.
	Version string

	// Git hash of the source code.
	GitHash string
}
