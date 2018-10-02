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
	"log"

	openzipkin "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/reporter/http"
	"go.opencensus.io/exporter/zipkin"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	yaml "gopkg.in/yaml.v2"
)

type zipkinConfig struct {
	Zipkin *struct {
		ServiceName      string `yaml:"service_name,omitempty"`
		Endpoint         string `yaml:"endpoint,omitempty"`
		LocalEndpointURI string `yaml:"local_endpoint,omitempty"`
	} `yaml:"zipkin,omitempty"`
}

type zipkinExporter struct{}

func (z *zipkinExporter) MakeExporters(config []byte) (se view.Exporter, te trace.Exporter, closer func()) {
	var c zipkinConfig
	if err := yaml.Unmarshal(config, &c); err != nil {
		log.Fatalf("Cannot unmarshal data: %v", err)
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
	te = zipkin.NewExporter(reporter, localEndpoint)
	closer = func() {
		if err := reporter.Close(); err != nil {
			log.Printf("Cannot close the Zipkin reporter: %v\n", err)
		}
	}
	return se, te, closer
}
