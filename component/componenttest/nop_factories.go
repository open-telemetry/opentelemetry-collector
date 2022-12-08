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

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/receiver"
)

// NopFactories returns a component.Factories with all nop factories.
func NopFactories() (component.Factories, error) {
	var factories component.Factories
	var err error

	//nolint:staticcheck
	if factories.Extensions, err = extension.MakeFactoryMap(NewNopExtensionFactory()); err != nil {
		return component.Factories{}, err
	}

	if factories.Receivers, err = receiver.MakeFactoryMap(NewNopReceiverFactory()); err != nil {
		return component.Factories{}, err
	}

	//nolint:staticcheck
	if factories.Exporters, err = exporter.MakeFactoryMap(NewNopExporterFactory()); err != nil {
		return component.Factories{}, err
	}

	if factories.Processors, err = component.MakeProcessorFactoryMap(NewNopProcessorFactory()); err != nil {
		return component.Factories{}, err
	}

	return factories, err
}
