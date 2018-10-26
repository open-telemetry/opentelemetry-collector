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
	"fmt"
	"sync"

	openzipkin "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/reporter/http"
	"go.opencensus.io/exporter/zipkin"
	"go.opencensus.io/trace"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	"github.com/census-instrumentation/opencensus-service/exporter"
)

type ZipkinConfig struct {
	ServiceName      string `yaml:"service_name,omitempty"`
	Endpoint         string `yaml:"endpoint,omitempty"`
	LocalEndpointURI string `yaml:"local_endpoint,omitempty"`
}

// zipkinExporter is a multiplexing exporter that spawns a new OpenCensus-Go Zipkin
// exporter per unique node encountered. This is because serviceNames per node define
// unique services, alongside their IPs. Also it is useful to intercept traffic from
// Zipkin servers and then transform them back to the final form when creating an
// OpenCensus spandata.
type zipkinExporter struct {
	// mu protects the fields below
	mu sync.Mutex

	defaultExporter         *zipkin.Exporter
	defaultServiceName      string
	defaultLocalEndpointURI string

	endpointURI string

	// The goal of this map is to multiplex and route
	// different serviceNames with various exporters
	serviceNameToExporter map[string]*zipkin.Exporter

	shutdownFns []func() error
}

const (
	DefaultZipkinEndpointHostPort = "localhost:9411"
	DefaultZipkinEndpointURL      = "http://" + DefaultZipkinEndpointHostPort + "/api/v2/spans"
)

func (zc *ZipkinConfig) EndpointURL() string {
	// If no endpoint was set, use the default Zipkin reporter URI.
	endpoint := DefaultZipkinEndpointURL
	if zc != nil && zc.Endpoint != "" {
		endpoint = zc.Endpoint
	}
	return endpoint
}

func ZipkinExportersFromYAML(config []byte) (tes []exporter.TraceExporter, doneFns []func() error, err error) {
	var cfg struct {
		Exporters *struct {
			Zipkin *ZipkinConfig `yaml:"zipkin"`
		} `yaml:"exporters"`
	}
	if err := yamlUnmarshal(config, &cfg); err != nil {
		return nil, nil, err
	}
	if cfg.Exporters == nil {
		return nil, nil, nil
	}

	zc := cfg.Exporters.Zipkin
	if zc == nil {
		return nil, nil, nil
	}

	serviceName := ""
	if zc.ServiceName != "" {
		serviceName = zc.ServiceName
	}
	localEndpointURI := "192.168.1.5:5454"
	if zc.LocalEndpointURI != "" {
		localEndpointURI = zc.LocalEndpointURI
	}
	endpoint := zc.EndpointURL()
	zle, err := newZipkinExporter(endpoint, serviceName, localEndpointURI)
	if err != nil {
		return nil, nil, fmt.Errorf("Cannot configure Zipkin exporter: %v", err)
	}
	tes = append(tes, zle)
	doneFns = append(doneFns, zle.stop)
	return
}

func newZipkinExporter(finalEndpointURI, defaultServiceName, defaultLocalEndpointURI string) (*zipkinExporter, error) {
	localEndpoint, err := openzipkin.NewEndpoint(defaultServiceName, defaultLocalEndpointURI)
	if err != nil {
		return nil, err
	}
	reporter := http.NewReporter(finalEndpointURI)
	defaultExporter := zipkin.NewExporter(reporter, localEndpoint)
	zle := &zipkinExporter{
		endpointURI:             finalEndpointURI,
		defaultExporter:         defaultExporter,
		defaultServiceName:      defaultServiceName,
		defaultLocalEndpointURI: defaultLocalEndpointURI,
		serviceNameToExporter:   make(map[string]*zipkin.Exporter),
	}

	// Ensure that we add the default reporter's Close functions
	zle.shutdownFns = append(zle.shutdownFns, reporter.Close)

	return zle, nil
}

// exporterForNode firstly tries to find a memoize OpenCensus-Go Zipkin exporter
// appropriate for this node. If it doesn't find any, it returns the default exporter
// that was created at start-time.
func (ze *zipkinExporter) exporterForNode(node *commonpb.Node) *zipkin.Exporter {
	ze.mu.Lock()
	defer ze.mu.Unlock()

	if node == nil {
		return ze.defaultExporter
	}

	serviceName := node.ServiceInfo.GetName()
	if serviceName == "" {
		serviceName = ze.defaultServiceName
	}

	// Make the unique key for this local endpoint/node: "serviceName" + "ipv4" + "ipv6" + "port"
	key := serviceName + node.Attributes["ipv4"] + node.Attributes["ipv6"] + node.Attributes["port"]
	if key == "" {
		return ze.defaultExporter
	}

	// Try looking up if we already created the exporter.
	if exp, ok := ze.serviceNameToExporter[key]; ok && exp != nil {
		return exp
	}

	localEndpointURI := ze.defaultLocalEndpointURI
	if ipv4 := node.Attributes["ipv4"]; ipv4 != "" {
		localEndpointURI = ipv4
	} else if ipv6 := node.Attributes["ipv6"]; ipv6 != "" {
		localEndpointURI = ipv6
	}

	// Otherwise freshly create that Zipkin Exporter.
	localEndpoint, err := openzipkin.NewEndpoint(serviceName, localEndpointURI)
	if err != nil {
		return ze.defaultExporter
	}

	reporter := http.NewReporter(ze.endpointURI)
	exporter := zipkin.NewExporter(reporter, localEndpoint)

	// Now memoize the created exporter for later use.
	ze.serviceNameToExporter[key] = exporter
	ze.shutdownFns = append(ze.shutdownFns, reporter.Close)

	return exporter
}

func (ze *zipkinExporter) stop() error {
	ze.mu.Lock()
	defer ze.mu.Unlock()

	for _, shutdownFn := range ze.shutdownFns {
		_ = shutdownFn()
	}

	return nil
}

func (ze *zipkinExporter) ExportSpanData(ctx context.Context, node *commonpb.Node, spandata ...*trace.SpanData) error {
	exporter := ze.exporterForNode(node)
	return exportSpans(ctx, node, "zipkin", exporter, spandata)
}
