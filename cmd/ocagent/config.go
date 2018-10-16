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
	"log"

	yaml "gopkg.in/yaml.v2"

	"github.com/census-instrumentation/opencensus-service/exporter"
	"github.com/census-instrumentation/opencensus-service/exporter/exporterparser"
)

const defaultOCInterceptorAddress = "localhost:55678"

type config struct {
	OpenCensusInterceptorConfig *interceptorConfig `yaml:"opencensus_interceptor"`
}

type interceptorConfig struct {
	// The address to which the OpenCensus interceptor will be bound and run on.
	Address string `yaml:"address"`
}

func (c *config) ocInterceptorAddress() string {
	if c == nil || c.OpenCensusInterceptorConfig == nil || c.OpenCensusInterceptorConfig.Address == "" {
		return defaultOCInterceptorAddress
	}
	return c.OpenCensusInterceptorConfig.Address
}

func parseOCAgentConfig(yamlBlob []byte) (*config, error) {
	var cfg config
	if err := yaml.Unmarshal(yamlBlob, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

type exporterParser func(yamlConfig []byte) (te []exporter.TraceExporter, err error)

// exportersFromYAMLConfig parses the config yaml payload and returns the respective exporters
func exportersFromYAMLConfig(config []byte) (traceExporters []exporter.TraceExporter, doneFns []func() error) {
	parseFns := []struct {
		name string
		fn   func([]byte) ([]exporter.TraceExporter, []func() error, error)
	}{
		{name: "datadog", fn: exporterparser.DatadogTraceExportersFromYAML},
		{name: "stackdriver", fn: exporterparser.StackdriverTraceExportersFromYAML},
		{name: "zipkin", fn: exporterparser.ZipkinExportersFromYAML},
	}

	for _, cfg := range parseFns {
		tes, tesDoneFns, err := cfg.fn(config)
		if err != nil {
			log.Fatalf("Failed to create config for %q: %v", cfg.name, err)
		}
		for _, te := range tes {
			if te != nil {
				traceExporters = append(traceExporters, te)
			}
		}
		for _, doneFn := range tesDoneFns {
			doneFns = append(doneFns, doneFn)
		}
	}
	return
}
