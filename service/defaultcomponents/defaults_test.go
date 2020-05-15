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

// Program otelcol is the OpenTelemetry Collector that collects stats
// and traces and exports to a configured backend.
package defaultcomponents

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/exporter/fileexporter"
	"go.opentelemetry.io/collector/exporter/jaegerexporter"
	"go.opentelemetry.io/collector/exporter/loggingexporter"
	"go.opentelemetry.io/collector/exporter/opencensusexporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/exporter/prometheusexporter"
	"go.opentelemetry.io/collector/exporter/zipkinexporter"
	"go.opentelemetry.io/collector/extension/healthcheckextension"
	"go.opentelemetry.io/collector/extension/pprofextension"
	"go.opentelemetry.io/collector/extension/zpagesextension"
	"go.opentelemetry.io/collector/processor/attributesprocessor"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/processor/memorylimiter"
	"go.opentelemetry.io/collector/processor/queuedprocessor"
	"go.opentelemetry.io/collector/processor/resourceprocessor"
	"go.opentelemetry.io/collector/processor/samplingprocessor/probabilisticsamplerprocessor"
	"go.opentelemetry.io/collector/processor/samplingprocessor/tailsamplingprocessor"
	"go.opentelemetry.io/collector/processor/spanprocessor"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver"
	"go.opentelemetry.io/collector/receiver/jaegerreceiver"
	"go.opentelemetry.io/collector/receiver/opencensusreceiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/prometheusreceiver"
	"go.opentelemetry.io/collector/receiver/vmmetricsreceiver"
	"go.opentelemetry.io/collector/receiver/zipkinreceiver"
)

func TestDefaultComponents(t *testing.T) {
	expectedExtensions := map[configmodels.Type]component.ExtensionFactory{
		"health_check": &healthcheckextension.Factory{},
		"pprof":        &pprofextension.Factory{},
		"zpages":       &zpagesextension.Factory{},
	}
	expectedReceivers := map[configmodels.Type]component.ReceiverFactoryBase{
		"jaeger":      &jaegerreceiver.Factory{},
		"zipkin":      &zipkinreceiver.Factory{},
		"prometheus":  &prometheusreceiver.Factory{},
		"opencensus":  &opencensusreceiver.Factory{},
		"otlp":        &otlpreceiver.Factory{},
		"vmmetrics":   &vmmetricsreceiver.Factory{},
		"hostmetrics": hostmetricsreceiver.NewFactory(),
	}
	expectedProcessors := map[configmodels.Type]component.ProcessorFactoryBase{
		"attributes":            &attributesprocessor.Factory{},
		"resource":              &resourceprocessor.Factory{},
		"queued_retry":          &queuedprocessor.Factory{},
		"batch":                 &batchprocessor.Factory{},
		"memory_limiter":        &memorylimiter.Factory{},
		"tail_sampling":         &tailsamplingprocessor.Factory{},
		"probabilistic_sampler": &probabilisticsamplerprocessor.Factory{},
		"span":                  &spanprocessor.Factory{},
	}
	expectedExporters := map[configmodels.Type]component.ExporterFactoryBase{
		"opencensus": &opencensusexporter.Factory{},
		"prometheus": &prometheusexporter.Factory{},
		"logging":    &loggingexporter.Factory{},
		"zipkin":     &zipkinexporter.Factory{},
		"jaeger":     &jaegerexporter.Factory{},
		"file":       &fileexporter.Factory{},
		"otlp":       &otlpexporter.Factory{},
	}

	factories, err := Components()
	assert.NoError(t, err)
	assert.Equal(t, expectedExtensions, factories.Extensions)
	assert.Equal(t, expectedReceivers, factories.Receivers)
	assert.Equal(t, expectedProcessors, factories.Processors)
	assert.Equal(t, expectedExporters, factories.Exporters)
}
