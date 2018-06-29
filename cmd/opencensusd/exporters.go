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

package main

import (
	"io/ioutil"
	"log"

	datadog "github.com/DataDog/opencensus-go-exporter-datadog"
	openzipkin "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/reporter/http"
	"go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/exporter/zipkin"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	yaml "gopkg.in/yaml.v2"
)

var (
	statsExporters []view.Exporter
	traceExporters []trace.Exporter
	closers        []func()
)

func exportViewData(vd *view.Data) {
	// TODO(jbd): Implement.
}

func exportSpanData(sd *trace.SpanData) {
	for _, exporter := range traceExporters {
		exporter.ExportSpan(sd)
	}
}

func flush() {
	for _, fn := range closers {
		fn()
	}
}

type config struct {
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

	Stackdriver struct {
		ProjectID     string `yaml:"project,omitempty"`
		EnableMetrics bool   `yaml:"enableMetrics,omitempty"`
		EnableTraces  bool   `yaml:"enableTraces,omitempty"`
	} `yaml:"stackdriver,omitempty"`

	Zipkin struct {
		Endpoint string `yaml:"endpoint,omitempty"`
	} `yaml:"zipkin,omitempty"`
}

func configureExporters() {
	const configFile = "config.yaml"
	conf, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Cannot read the %v file: %v", configFile, err)
	}
	var c config
	if err := yaml.Unmarshal(conf, &c); err != nil {
		log.Fatalf("Cannot unmarshal data: %v", err)
	}

	// Configure Zipkin.
	if endpoint := c.Zipkin.Endpoint; endpoint != "" {
		// TODO(jbd): Propagate service name, hostport and more metadata from each node.
		localEndpoint, err := openzipkin.NewEndpoint("server", "")
		if err != nil {
			log.Fatalf("Cannot configure Zipkin exporter: %v", err)
		}
		reporter := http.NewReporter(endpoint)
		exporter := zipkin.NewExporter(reporter, localEndpoint)
		traceExporters = append(traceExporters, exporter)

		closers = append(closers, func() {
			if err := reporter.Close(); err != nil {
				log.Printf("Cannot close the Zipkin reporter: %v\n", err)
			}
		})
	}

	// Configure Stackdriver.
	if s := c.Stackdriver; s.EnableMetrics || s.EnableTraces {
		// TODO(jbd): Add monitored resources.
		if s.ProjectID == "" {
			log.Fatal("Stackdriver config requires a project ID")
		}
		exporter, err := stackdriver.NewExporter(stackdriver.Options{
			ProjectID: s.ProjectID,
		})
		if err != nil {
			log.Fatalf("Cannot configure Stackdriver exporter: %v", err)
		}
		if s.EnableMetrics {
			statsExporters = append(statsExporters, exporter)
		}
		if s.EnableTraces {
			traceExporters = append(traceExporters, exporter)
		}
		closers = append(closers, exporter.Flush)
	}

	// Configure Datadog.
	if d := c.Datadog; d.EnableMetrics || d.EnableTraces {
		// TODO(jbd): Create new exporter for each service name.
		exporter := datadog.NewExporter(datadog.Options{
			Namespace: d.Namespace,
			TraceAddr: d.TraceAddr,
			StatsAddr: d.MetricsAddr,
			Tags:      d.Tags,
		})
		if d.EnableMetrics {
			statsExporters = append(statsExporters, exporter)
		}
		if d.EnableTraces {
			traceExporters = append(traceExporters, exporter)
		}
		closers = append(closers, exporter.Stop)
	}
}
