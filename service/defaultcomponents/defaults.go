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

// Package defaultcomponents composes the default set of components used by the otel service
package defaultcomponents

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/exporter/fileexporter"
	"go.opentelemetry.io/collector/exporter/jaegerexporter"
	"go.opentelemetry.io/collector/exporter/kafkaexporter"
	"go.opentelemetry.io/collector/exporter/loggingexporter"
	"go.opentelemetry.io/collector/exporter/opencensusexporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/exporter/prometheusexporter"
	"go.opentelemetry.io/collector/exporter/prometheusremotewriteexporter"
	"go.opentelemetry.io/collector/exporter/zipkinexporter"
	"go.opentelemetry.io/collector/extension/fluentbitextension"
	"go.opentelemetry.io/collector/extension/healthcheckextension"
	"go.opentelemetry.io/collector/extension/pprofextension"
	"go.opentelemetry.io/collector/extension/zpagesextension"
	"go.opentelemetry.io/collector/processor/attributesprocessor"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/processor/filterprocessor"
	"go.opentelemetry.io/collector/processor/memorylimiter"
	"go.opentelemetry.io/collector/processor/queuedprocessor"
	"go.opentelemetry.io/collector/processor/resourceprocessor"
	"go.opentelemetry.io/collector/processor/samplingprocessor/probabilisticsamplerprocessor"
	"go.opentelemetry.io/collector/processor/spanprocessor"
	"go.opentelemetry.io/collector/receiver/fluentforwardreceiver"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver"
	"go.opentelemetry.io/collector/receiver/jaegerreceiver"
	"go.opentelemetry.io/collector/receiver/kafkareceiver"
	"go.opentelemetry.io/collector/receiver/opencensusreceiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/prometheusreceiver"
	"go.opentelemetry.io/collector/receiver/zipkinreceiver"
)

// Components returns the default set of components used by the
// OpenTelemetry collector.
func Components() (
	component.Factories,
	error,
) {
	var errs []error

	extensions, err := component.MakeExtensionFactoryMap(
		healthcheckextension.NewFactory(),
		pprofextension.NewFactory(),
		zpagesextension.NewFactory(),
		fluentbitextension.NewFactory(),
	)
	if err != nil {
		errs = append(errs, err)
	}

	receivers, err := component.MakeReceiverFactoryMap(
		jaegerreceiver.NewFactory(),
		fluentforwardreceiver.NewFactory(),
		zipkinreceiver.NewFactory(),
		prometheusreceiver.NewFactory(),
		opencensusreceiver.NewFactory(),
		otlpreceiver.NewFactory(),
		hostmetricsreceiver.NewFactory(),
		kafkareceiver.NewFactory(),
	)
	if err != nil {
		errs = append(errs, err)
	}

	exporters, err := component.MakeExporterFactoryMap(
		opencensusexporter.NewFactory(),
		prometheusexporter.NewFactory(),
		prometheusremotewriteexporter.NewFactory(),
		loggingexporter.NewFactory(),
		zipkinexporter.NewFactory(),
		jaegerexporter.NewFactory(),
		fileexporter.NewFactory(),
		otlpexporter.NewFactory(),
		otlphttpexporter.NewFactory(),
		kafkaexporter.NewFactory(),
	)
	if err != nil {
		errs = append(errs, err)
	}

	processors, err := component.MakeProcessorFactoryMap(
		attributesprocessor.NewFactory(),
		resourceprocessor.NewFactory(),
		queuedprocessor.NewFactory(),
		batchprocessor.NewFactory(),
		memorylimiter.NewFactory(),
		probabilisticsamplerprocessor.NewFactory(),
		spanprocessor.NewFactory(),
		filterprocessor.NewFactory(),
	)
	if err != nil {
		errs = append(errs, err)
	}

	factories := component.Factories{
		Extensions: extensions,
		Receivers:  receivers,
		Processors: processors,
		Exporters:  exporters,
	}

	return factories, componenterror.CombineErrors(errs)
}
