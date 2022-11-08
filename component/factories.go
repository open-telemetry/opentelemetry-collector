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

package component // import "go.opentelemetry.io/collector/component"

import (
	"fmt"
)

// Factories struct holds in a single type all component factories that
// can be handled by the Config.
type Factories struct {
	// Receivers maps receiver type names in the config to the respective factory.
	Receivers map[Type]ReceiverFactory

	// Processors maps processor type names in the config to the respective factory.
	Processors map[Type]ProcessorFactory

	// Exporters maps exporter type names in the config to the respective factory.
	Exporters map[Type]ExporterFactory

	// Extensions maps extension type names in the config to the respective factory.
	Extensions map[Type]ExtensionFactory
}

// MakeReceiverFactoryMap takes a list of receiver factories and returns a map
// with factory type as keys. It returns a non-nil error when more than one factories
// have the same type.
func MakeReceiverFactoryMap(factories ...ReceiverFactory) (map[Type]ReceiverFactory, error) {
	fMap := map[Type]ReceiverFactory{}
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
func MakeProcessorFactoryMap(factories ...ProcessorFactory) (map[Type]ProcessorFactory, error) {
	fMap := map[Type]ProcessorFactory{}
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
func MakeExporterFactoryMap(factories ...ExporterFactory) (map[Type]ExporterFactory, error) {
	fMap := map[Type]ExporterFactory{}
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
func MakeExtensionFactoryMap(factories ...ExtensionFactory) (map[Type]ExtensionFactory, error) {
	fMap := map[Type]ExtensionFactory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate extension factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
