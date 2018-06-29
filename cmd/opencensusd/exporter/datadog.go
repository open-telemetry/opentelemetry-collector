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

package exporter

import (
	"log"

	datadog "github.com/DataDog/opencensus-go-exporter-datadog"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	yaml "gopkg.in/yaml.v2"
)

type dataDogConfig struct {
	Datadog struct {
		// Namespace specifies the namespaces to which metric keys are appended.
		Namespace string `yaml:"namespace,omitempty"`

		// TraceAddr specifies the host[:port] address of the Datadog Trace Agent.
		// It defaults to localhost:8126.
		TraceAddr string `yaml:"traceAddr,omitempty"`

		// MetricsAddr specifies the host[:port] address for DogStatsD. It defaults
		// to localhost:8125.
		MetricsAddr string `yaml:"metricsAddr,omitempty"`

		// Tags specifies a set of global tags to attach to each metric.
		Tags []string `yaml:"tags,omitempty"`

		EnableMetrics bool `yaml:"enableMetrics,omitempty"`
		EnableTraces  bool `yaml:"enableTraces,omitempty"`
	} `yaml:"datadog,omitempty"`
}

type datadogExporter struct{}

func (d *datadogExporter) MakeExporters(config []byte) (se view.Exporter, te trace.Exporter, closer func()) {
	var c dataDogConfig
	if err := yaml.Unmarshal(config, &c); err != nil {
		log.Fatalf("Cannot unmarshal data: %v", err)
	}
	if c := c.Datadog; c.EnableMetrics || c.EnableTraces {
		// TODO(jbd): Create new exporter for each service name.
		de := datadog.NewExporter(datadog.Options{
			Namespace: c.Namespace,
			TraceAddr: c.TraceAddr,
			StatsAddr: c.MetricsAddr,
			Tags:      c.Tags,
		})
		if c.EnableMetrics {
			se = de
		}
		if c.EnableTraces {
			te = de
		}
		closer = de.Stop
	}
	return se, te, closer
}
