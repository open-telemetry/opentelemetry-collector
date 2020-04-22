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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/component"
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
	"github.com/open-telemetry/opentelemetry-collector/processor/attributesprocessor"
	"github.com/open-telemetry/opentelemetry-collector/processor/batchprocessor"
	"github.com/open-telemetry/opentelemetry-collector/processor/memorylimiter"
	"github.com/open-telemetry/opentelemetry-collector/processor/queuedprocessor"
	"github.com/open-telemetry/opentelemetry-collector/processor/samplingprocessor/probabilisticsamplerprocessor"
	"github.com/open-telemetry/opentelemetry-collector/processor/samplingprocessor/tailsamplingprocessor"
	"github.com/open-telemetry/opentelemetry-collector/processor/spanprocessor"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/jaegerreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/opencensusreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/otlpreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/prometheusreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/vmmetricsreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/zipkinreceiver"
)

func TestDefaultComponents(t *testing.T) {
	expectedExtensions := map[string]component.ExtensionFactory{
		"health_check": &healthcheckextension.Factory{},
		"pprof":        &pprofextension.Factory{},
		"zpages":       &zpagesextension.Factory{},
	}
	expectedReceivers := map[string]component.ReceiverFactoryBase{
		"jaeger":      &jaegerreceiver.Factory{},
		"zipkin":      &zipkinreceiver.Factory{},
		"prometheus":  &prometheusreceiver.Factory{},
		"opencensus":  &opencensusreceiver.Factory{},
		"otlp":        &otlpreceiver.Factory{},
		"vmmetrics":   &vmmetricsreceiver.Factory{},
		"hostmetrics": hostmetricsreceiver.NewFactory(),
	}
	expectedProcessors := map[string]component.ProcessorFactoryBase{
		"attributes":            &attributesprocessor.Factory{},
		"queued_retry":          &queuedprocessor.Factory{},
		"batch":                 &batchprocessor.Factory{},
		"memory_limiter":        &memorylimiter.Factory{},
		"tail_sampling":         &tailsamplingprocessor.Factory{},
		"probabilistic_sampler": &probabilisticsamplerprocessor.Factory{},
		"span":                  &spanprocessor.Factory{},
	}
	expectedExporters := map[string]component.ExporterFactoryBase{
		"opencensus": &opencensusexporter.Factory{},
		"prometheus": &prometheusexporter.Factory{},
		"logging":    &loggingexporter.Factory{},
		"zipkin":     &zipkinexporter.Factory{},
		"jaeger":     &jaegerexporter.Factory{},
		"file":       &fileexporter.Factory{},
		"otlp":       &otlpexporter.Factory{},
	}

	factories, err := Components()
	fmt.Println(err)
	assert.Nil(t, err)
	assert.Equal(t, expectedExtensions, factories.Extensions)
	assert.Equal(t, expectedReceivers, factories.Receivers)
	assert.Equal(t, expectedProcessors, factories.Processors)
	assert.Equal(t, expectedExporters, factories.Exporters)
}
