// Copyright 2018, OpenCensus Authors
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

package exporterparser

import (
	"context"

	datadog "github.com/DataDog/opencensus-go-exporter-datadog"
	"go.opencensus.io/trace"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	"github.com/census-instrumentation/opencensus-service/exporter"
)

type dataDogConfig struct {
	Datadog *struct {
		// Namespace specifies the namespaces to which metric keys are appended.
		Namespace string `yaml:"namespace,omitempty"`

		// TraceAddr specifies the host[:port] address of the Datadog Trace Agent.
		// It defaults to localhost:8126.
		TraceAddr string `yaml:"trace_addr,omitempty"`

		// MetricsAddr specifies the host[:port] address for DogStatsD. It defaults
		// to localhost:8125.
		MetricsAddr string `yaml:"metrics_addr,omitempty"`

		// Tags specifies a set of global tags to attach to each metric.
		Tags []string `yaml:"tags,omitempty"`

		EnableTracing bool `yaml:"enable_tracing,omitempty"`
	} `yaml:"datadog,omitempty"`
}

type datadogExporter struct {
	exporter *datadog.Exporter
}

func DatadogTraceExportersFromYAML(config []byte) (tes []exporter.TraceExporter, doneFns []func() error, err error) {
	var c dataDogConfig
	if err := yamlUnmarshal(config, &c); err != nil {
		return nil, nil, err
	}

	dc := c.Datadog
	if dc == nil {
		return nil, nil, nil
	}
	if !dc.EnableTracing {
		return nil, nil, nil
	}

	// TODO(jbd): Create new exporter for each service name.
	de := datadog.NewExporter(datadog.Options{
		Namespace: dc.Namespace,
		TraceAddr: dc.TraceAddr,
		StatsAddr: dc.MetricsAddr,
		Tags:      dc.Tags,
	})
	doneFns = append(doneFns, func() error {
		de.Stop()
		return nil
	})
	tes = append(tes, &datadogExporter{exporter: de})
	return
}

func (dde *datadogExporter) ExportSpanData(ctx context.Context, node *commonpb.Node, spandata ...*trace.SpanData) error {
	// TODO: Examine the Datadog exporter to see
	// if trace.ExportSpan was constraining and if perhaps the
	// upload can use the context and information from the Node.
	return exportSpans(ctx, node, "datadog", dde.exporter, spandata)
}
