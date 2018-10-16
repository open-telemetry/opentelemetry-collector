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
	"log"

	openzipkin "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/reporter/http"
	"go.opencensus.io/exporter/zipkin"
	"go.opencensus.io/trace"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	"github.com/census-instrumentation/opencensus-service/exporter"
)

type zipkinConfig struct {
	Zipkin *struct {
		ServiceName      string `yaml:"service_name,omitempty"`
		Endpoint         string `yaml:"endpoint,omitempty"`
		LocalEndpointURI string `yaml:"local_endpoint,omitempty"`
	} `yaml:"zipkin,omitempty"`
}

type zipkinExporter struct {
	exporter *zipkin.Exporter
}

func ZipkinExportersFromYAML(config []byte) (tes []exporter.TraceExporter, doneFns []func() error, err error) {
	var c zipkinConfig
	if err := yamlUnmarshal(config, &c); err != nil {
		return nil, nil, err
	}

	if c.Zipkin == nil {
		return nil, nil, nil
	}
	zc := c.Zipkin
	endpoint := "http://localhost:9411/api/v2/spans"
	if zc.Endpoint != "" {
		endpoint = zc.Endpoint
	}
	serviceName := ""
	if zc.ServiceName != "" {
		serviceName = zc.ServiceName
	}
	localEndpointURI := "192.168.1.5:5454"
	if zc.LocalEndpointURI != "" {
		localEndpointURI = zc.LocalEndpointURI
	}
	// TODO(jbd): Propagate hostport and more metadata from each node.
	localEndpoint, err := openzipkin.NewEndpoint(serviceName, localEndpointURI)
	if err != nil {
		log.Fatalf("Cannot configure Zipkin exporter: %v", err)
	}

	reporter := http.NewReporter(endpoint)
	ze := zipkin.NewExporter(reporter, localEndpoint)
	tes = append(tes, &zipkinExporter{exporter: ze})
	doneFns = append(doneFns, reporter.Close)
	return
}

func (ze *zipkinExporter) ExportSpanData(ctx context.Context, node *commonpb.Node, spandata ...*trace.SpanData) error {
	// TODO: Examine "contrib.go.opencensus.io/exporter/zipkin" to see
	// if trace.ExportSpan was constraining and if perhaps the Zipkin
	// upload can use the context and information from the Node.
	return exportSpans(ze.exporter, spandata)
}
