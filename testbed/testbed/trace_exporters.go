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

package testbed

import (
	"fmt"

	"contrib.go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/trace"
)

// TraceExporter defines the interface that allows sending span data. This is an interface
// that must be implemented by all protocols that want to be used in LoadGenerator.
// Note the terminology: testbed.TraceExporter is something that sends data to Collector
// and the corresponding entity in the Collector is a receiver.
type TraceExporter interface {
	ExportSpans(spans []*trace.SpanData)
	Flush()

	// Return the port to which this exporter will send data.
	GetCollectorPort() int

	// Generate a config string to place in receiver part of collector config
	// so that it can receive data from this exporter.
	GenConfigYAMLStr() string

	// Return protocol name to use in collector config pipeline.
	ProtocolName() string
}

type jaegerExporter struct {
	exporter *jaeger.Exporter
	port     int
}

// Create a new Jaeger protocol exporter.
func NewJaegerExporter(port int) *jaegerExporter {
	opts := jaeger.Options{
		// Use standard URL for Jaeger.
		CollectorEndpoint: fmt.Sprintf("http://localhost:%d/api/traces", port),
		Process: jaeger.Process{
			ServiceName: "load-generator",
		},
	}

	exporter, err := jaeger.NewExporter(opts)
	if err != nil {
		return nil
	}

	return &jaegerExporter{exporter: exporter, port: port}
}

func (je *jaegerExporter) ExportSpans(spans []*trace.SpanData) {
	for _, span := range spans {
		je.exporter.ExportSpan(span)
	}
}

func (je *jaegerExporter) Flush() {
	je.exporter.Flush()
}

func (je *jaegerExporter) GenConfigYAMLStr() string {
	// Note that this generates a receiver config for agent.
	// We only need to enable thrift-http protocol because that's what we use in tests.
	// Due to bug in Jaeger receiver (https://github.com/open-telemetry/opentelemetry-collector/issues/445)
	// which makes it impossible to disable protocols that we don't need to receive on we
	// have to use fake ports for all endpoints except thrift-http, otherwise it is
	// impossible to start the Collector because the standard ports for those protocols
	// are already listened by mock Jaeger backend that is part of the tests.
	// As soon as the bug is fixed remove the endpoints and use "disabled: true" setting
	// instead.
	return fmt.Sprintf(`
  jaeger:
    protocols:
      grpc:
        endpoint: "localhost:8371"
      thrift-tchannel:
        endpoint: "localhost:8372"
      thrift-compact:
        endpoint: "localhost:8373"
      thrift-binary:
        endpoint: "localhost:8374"
      thrift-http:
        endpoint: "localhost:%d"`, je.port)
}

func (je *jaegerExporter) GetCollectorPort() int {
	return je.port
}

func (je *jaegerExporter) ProtocolName() string {
	return "jaeger"
}
