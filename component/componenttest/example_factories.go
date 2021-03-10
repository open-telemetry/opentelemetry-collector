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

package componenttest

import (
	"go.opentelemetry.io/collector/component"
)

// ExampleComponents registers example factories. This is only used by tests.
func ExampleComponents() (
	factories component.Factories,
	err error,
) {
	if factories.Extensions, err = component.MakeExtensionFactoryMap(&ExampleExtensionFactory{}); err != nil {
		return
	}

	factories.Receivers, err = component.MakeReceiverFactoryMap(
		&ExampleReceiverFactory{},
	)
	if err != nil {
		return
	}

	factories.Exporters, err = component.MakeExporterFactoryMap(&ExampleExporterFactory{})
	if err != nil {
		return
	}

	factories.Processors, err = component.MakeProcessorFactoryMap(&ExampleProcessorFactory{})

	return
}
