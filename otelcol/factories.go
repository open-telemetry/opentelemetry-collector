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

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
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
}
