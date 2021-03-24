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
	// Components lifecycle goes through the following phases:
	//
	// 1. Creation. The component is created using the factory, via Create* call.
	// 2. Start. The component's Start() method is called.
	// 3. Running. The component is up and running.
	// 4. Shutdown. The component's Shutdown() method is called, the lifecycle is complete.
	//
	// Once the lifecycle is complete it may be repeated, in which case a new component
	// is created and goes through the lifecycle.

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

	// Shutdown is invoked during service shutdown. After Shutdown() is called, if the component accept data in
	// any way, it should not accept it anymore.
	//
	// If there are any background operations running by the component they must be aborted as soon as possible.
	// Remember that if you started any long-running background operation from the Start() method that operation
	// must be also cancelled. If there are any buffer in the component, it should be cleared and the data sent
	// immediately to the next component.
	//
	// Once the Shutdown() method returns the component's lifecycle is completed. No other
	// methods of the component are called after that. If necessary a new component with
	// the same or different configuration may be created and started (this may happen
	// for example if we want to restart the component).
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
