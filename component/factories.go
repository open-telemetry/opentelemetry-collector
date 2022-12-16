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

// Deprecated: [v0.68.0] use otelcol.Factories.
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

// Deprecated: [v0.68.0] use processor.MakeFactoryMap
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
