// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package defaults composes the default set of components used by the otel service
package defaults

import (
	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config"
	"github.com/open-telemetry/opentelemetry-collector/exporter/fileexporter"
	"github.com/open-telemetry/opentelemetry-collector/exporter/jaegerexporter"
	"github.com/open-telemetry/opentelemetry-collector/exporter/loggingexporter"
	"github.com/open-telemetry/opentelemetry-collector/exporter/opencensusexporter"
	"github.com/open-telemetry/opentelemetry-collector/exporter/otlpexporter"
	"github.com/open-telemetry/opentelemetry-collector/exporter/prometheusexporter"
	"github.com/open-telemetry/opentelemetry-collector/exporter/zipkinexporter"
	"github.com/open-telemetry/opentelemetry-collector/extension/healthcheckextension"
	"github.com/open-telemetry/opentelemetry-collector/extension/pprofextension"
	"github.com/open-telemetry/opentelemetry-collector/extension/zpagesextension"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
	"github.com/open-telemetry/opentelemetry-collector/processor/attributesprocessor"
	"github.com/open-telemetry/opentelemetry-collector/processor/batchprocessor"
	"github.com/open-telemetry/opentelemetry-collector/processor/memorylimiter"
	"github.com/open-telemetry/opentelemetry-collector/processor/queuedprocessor"
	"github.com/open-telemetry/opentelemetry-collector/processor/samplingprocessor/probabilisticsamplerprocessor"
	"github.com/open-telemetry/opentelemetry-collector/processor/samplingprocessor/tailsamplingprocessor"
	"github.com/open-telemetry/opentelemetry-collector/processor/spanprocessor"
	"github.com/open-telemetry/opentelemetry-collector/receiver/jaegerreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/opencensusreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/otlpreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/prometheusreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/vmmetricsreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/zipkinreceiver"
)

// Components returns the default set of components used by the
// OpenTelemetry collector.
func Components() (
	config.Factories,
	error,
) {
	errs := []error{}

	extensions, err := component.MakeExtensionFactoryMap(
		&healthcheckextension.Factory{},
		&pprofextension.Factory{},
		&zpagesextension.Factory{},
	)
	if err != nil {
		errs = append(errs, err)
	}

	receivers, err := component.MakeReceiverFactoryMap(
		&jaegerreceiver.Factory{},
		&zipkinreceiver.Factory{},
		&prometheusreceiver.Factory{},
		&opencensusreceiver.Factory{},
		&otlpreceiver.Factory{},
		&vmmetricsreceiver.Factory{},
	)
	if err != nil {
		errs = append(errs, err)
	}

	exporters, err := component.MakeExporterFactoryMap(
		&opencensusexporter.Factory{},
		&prometheusexporter.Factory{},
		&loggingexporter.Factory{},
		&zipkinexporter.Factory{},
		&jaegerexporter.Factory{},
		&fileexporter.Factory{},
		&otlpexporter.Factory{},
	)
	if err != nil {
		errs = append(errs, err)
	}

	processors, err := component.MakeProcessorFactoryMap(
		&attributesprocessor.Factory{},
		&queuedprocessor.Factory{},
		&batchprocessor.Factory{},
		&memorylimiter.Factory{},
		&tailsamplingprocessor.Factory{},
		&probabilisticsamplerprocessor.Factory{},
		&spanprocessor.Factory{},
	)
	if err != nil {
		errs = append(errs, err)
	}

	factories := config.Factories{
		Extensions: extensions,
		Receivers:  receivers,
		Processors: processors,
		Exporters:  exporters,
	}

	return factories, oterr.CombineErrors(errs)
}
