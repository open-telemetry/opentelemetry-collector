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

package testcomponents

import (
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/loggingexporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/extension/ballastextension"
	"go.opentelemetry.io/collector/extension/zpagesextension"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
)

// ExampleComponents registers example factories. This is only used by tests.
func ExampleComponents() (
	factories component.Factories,
	err error,
) {
	if factories.Extensions, err = component.MakeExtensionFactoryMap(ExampleExtensionFactory); err != nil {
		return
	}

	if factories.Receivers, err = component.MakeReceiverFactoryMap(ExampleReceiverFactory); err != nil {
		return
	}

	if factories.Exporters, err = component.MakeExporterFactoryMap(ExampleExporterFactory); err != nil {
		return
	}

	factories.Processors, err = component.MakeProcessorFactoryMap(ExampleProcessorFactory)

	return
}

// DefaultComponents returns the default set of components used by the OpenTelemetry collector.
func DefaultComponents() (
	component.Factories,
	error,
) {
	var errs error

	extensions, err := component.MakeExtensionFactoryMap(
		zpagesextension.NewFactory(),
		ballastextension.NewFactory(),
	)
	errs = multierr.Append(errs, err)

	receivers, err := component.MakeReceiverFactoryMap(
		otlpreceiver.NewFactory(),
	)
	errs = multierr.Append(errs, err)

	exporters, err := component.MakeExporterFactoryMap(
		loggingexporter.NewFactory(),
		otlpexporter.NewFactory(),
		otlphttpexporter.NewFactory(),
	)
	errs = multierr.Append(errs, err)

	processors, err := component.MakeProcessorFactoryMap(
		batchprocessor.NewFactory(),
		memorylimiterprocessor.NewFactory(),
	)
	errs = multierr.Append(errs, err)

	factories := component.Factories{
		Extensions: extensions,
		Receivers:  receivers,
		Processors: processors,
		Exporters:  exporters,
	}

	return factories, errs
}
