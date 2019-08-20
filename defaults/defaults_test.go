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

// Program otelsvc is the Open Telemetry Service that collects stats
// and traces and exports to a configured backend.
package defaults

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-service/exporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/jaeger/jaegergrpcexporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/jaeger/jaegerthrifthttpexporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/loggingexporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/opencensusexporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/prometheusexporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/zipkinexporter"
	"github.com/open-telemetry/opentelemetry-service/processor"
	"github.com/open-telemetry/opentelemetry-service/processor/addattributesprocessor"
	"github.com/open-telemetry/opentelemetry-service/processor/attributekeyprocessor"
	"github.com/open-telemetry/opentelemetry-service/processor/nodebatcherprocessor"
	"github.com/open-telemetry/opentelemetry-service/processor/queuedprocessor"
	"github.com/open-telemetry/opentelemetry-service/processor/tailsamplingprocessor"
	"github.com/open-telemetry/opentelemetry-service/receiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/jaegerreceiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/opencensusreceiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/prometheusreceiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/vmmetricsreceiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/zipkinreceiver"
)

func TestDefaultComponents(t *testing.T) {
	expectedReceivers := map[string]receiver.Factory{
		"jaeger":     &jaegerreceiver.Factory{},
		"zipkin":     &zipkinreceiver.Factory{},
		"prometheus": &prometheusreceiver.Factory{},
		"opencensus": &opencensusreceiver.Factory{},
		"vmmetrics":  &vmmetricsreceiver.Factory{},
	}
	expectedProcessors := map[string]processor.Factory{
		"add-attributes": &addattributesprocessor.Factory{},
		"attribute-key":  &attributekeyprocessor.Factory{},
		"queued-retry":   &queuedprocessor.Factory{},
		"batch":          &nodebatcherprocessor.Factory{},
		"tail-sampling":  &tailsamplingprocessor.Factory{},
	}
	expectedExporters := map[string]exporter.Factory{
		"opencensus":         &opencensusexporter.Factory{},
		"prometheus":         &prometheusexporter.Factory{},
		"logging":            &loggingexporter.Factory{},
		"zipkin":             &zipkinexporter.Factory{},
		"jaeger-grpc":        &jaegergrpcexporter.Factory{},
		"jaeger-thrift-http": &jaegerthrifthttpexporter.Factory{},
	}

	receivers, processors, exporters, err := Components()
	fmt.Println(err)
	assert.Nil(t, err)
	assert.Equal(t, expectedReceivers, receivers)
	assert.Equal(t, expectedProcessors, processors)
	assert.Equal(t, expectedExporters, exporters)
}
