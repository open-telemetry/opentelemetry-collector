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

package service // import "go.opentelemetry.io/collector/component"

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
)

// Factories struct holds in a single type all component factories that
// can be handled by the Config.
type Factories struct {
	// Receivers maps receiver type names in the config to the respective factory.
	Receivers map[component.Type]component.ReceiverFactory

	// Processors maps processor type names in the config to the respective factory.
	Processors map[component.Type]component.ProcessorFactory

	// Exporters maps exporter type names in the config to the respective factory.
	Exporters map[component.Type]exporter.Factory

	// Extensions maps extension type names in the config to the respective factory.
	Extensions map[component.Type]extension.Factory
}

// MakeReceiverFactoryMap takes a list of receiver factories and returns a map
// with factory type as keys. It returns a non-nil error when more than one factories
// have the same type.
func MakeReceiverFactoryMap(factories ...component.ReceiverFactory) (map[component.Type]component.ReceiverFactory, error) {
	fMap := map[component.Type]component.ReceiverFactory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate receiver factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}

// MakeProcessorFactoryMap takes a list of processor factories and returns a map
// with factory type as keys. It returns a non-nil error when more than one factories
// have the same type.
func MakeProcessorFactoryMap(factories ...component.ProcessorFactory) (map[component.Type]component.ProcessorFactory, error) {
	fMap := map[component.Type]component.ProcessorFactory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate processor factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}

// MakeExporterFactoryMap takes a list of exporter factories and returns a map
// with factory type as keys. It returns a non-nil error when more than one factories
// have the same type.
func MakeExporterFactoryMap(factories ...exporter.Factory) (map[component.Type]exporter.Factory, error) {
	fMap := map[component.Type]exporter.Factory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate exporter factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}

// MakeExtensionFactoryMap takes a list of extension factories and returns a map
// with factory type as keys. It returns a non-nil error when more than one factories
// have the same type.
func MakeExtensionFactoryMap(factories ...extension.Factory) (map[component.Type]extension.Factory, error) {
	fMap := map[component.Type]extension.Factory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate extension factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
