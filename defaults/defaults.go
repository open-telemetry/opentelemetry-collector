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
	"github.com/open-telemetry/opentelemetry-service/exporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/jaeger/jaegergrpcexporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/jaeger/jaegerthrifthttpexporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/loggingexporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/opencensusexporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/prometheusexporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/zipkinexporter"
	"github.com/open-telemetry/opentelemetry-service/oterr"
	"github.com/open-telemetry/opentelemetry-service/processor"
	"github.com/open-telemetry/opentelemetry-service/processor/attributesprocessor"
	"github.com/open-telemetry/opentelemetry-service/processor/nodebatcherprocessor"
	"github.com/open-telemetry/opentelemetry-service/processor/probabilisticsamplerprocessor"
	"github.com/open-telemetry/opentelemetry-service/processor/queuedprocessor"
	"github.com/open-telemetry/opentelemetry-service/processor/tailsamplingprocessor"
	"github.com/open-telemetry/opentelemetry-service/receiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/jaegerreceiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/opencensusreceiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/prometheusreceiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/vmmetricsreceiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/zipkinreceiver"
)

// Components returns the default set of components used by the
// opentelemetry service
func Components() (
	map[string]receiver.Factory,
	map[string]processor.Factory,
	map[string]exporter.Factory,
	error,
) {
	errs := []error{}
	receivers, err := receiver.Build(
		&jaegerreceiver.Factory{},
		&zipkinreceiver.Factory{},
		&prometheusreceiver.Factory{},
		&opencensusreceiver.Factory{},
		&vmmetricsreceiver.Factory{},
	)
	if err != nil {
		errs = append(errs, err)
	}

	exporters, err := exporter.Build(
		&opencensusexporter.Factory{},
		&prometheusexporter.Factory{},
		&loggingexporter.Factory{},
		&zipkinexporter.Factory{},
		&jaegergrpcexporter.Factory{},
		&jaegerthrifthttpexporter.Factory{},
	)
	if err != nil {
		errs = append(errs, err)
	}

	processors, err := processor.Build(
		&attributesprocessor.Factory{},
		&queuedprocessor.Factory{},
		&nodebatcherprocessor.Factory{},
		&tailsamplingprocessor.Factory{},
		&probabilisticsamplerprocessor.Factory{},
	)
	if err != nil {
		errs = append(errs, err)
	}
	return receivers, processors, exporters, oterr.CombineErrors(errs)
}
