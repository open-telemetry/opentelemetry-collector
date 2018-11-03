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
	"fmt"
	"log"
	"net"
	"net/url"
	"reflect"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/census-instrumentation/opencensus-service/exporter"
	"github.com/census-instrumentation/opencensus-service/exporter/exporterparser"
)

// We expect the configuration.yaml file to look like this:
//
//  interceptors:
//      opencensus:
//          port: <port>
//      zipkin:
//          reporter: <address>
//
//  exporters:
//      stackdriver:
//          project: <project_id>
//          enable_tracing: true
//      zipkin:
//          endpoint: "http://localhost:9411/api/v2/spans"
//
//  zpages:
//      port: 55679

const (
	defaultOCInterceptorAddress = ":55678"
	defaultZPagesPort           = 55679
)

type config struct {
	Interceptors *interceptors `yaml:"interceptors"`
	ZPages       *zPagesConfig `yaml:"zpages"`
}

type interceptors struct {
	OpenCensus *interceptorConfig `yaml:"opencensus"`
	Zipkin     *interceptorConfig `yaml:"zipkin"`
}

type interceptorConfig struct {
	// The address to which the OpenCensus interceptor will be bound and run on.
	Address string `yaml:"address"`
}

type zPagesConfig struct {
	Disabled bool `yaml:"disabled"`
	Port     int  `yaml:"port"`
}

func (c *config) ocInterceptorAddress() string {
	if c == nil || c.Interceptors == nil {
		return defaultOCInterceptorAddress
	}
	inCfg := c.Interceptors
	if inCfg.OpenCensus == nil || inCfg.OpenCensus.Address == "" {
		return defaultOCInterceptorAddress
	}
	return inCfg.OpenCensus.Address
}

func (c *config) zPagesDisabled() bool {
	if c == nil {
		return true
	}
	return c.ZPages != nil && c.ZPages.Disabled
}

func (c *config) zPagesPort() (int, bool) {
	if c.zPagesDisabled() {
		return -1, false
	}
	port := defaultZPagesPort
	if c != nil && c.ZPages != nil && c.ZPages.Port > 0 {
		port = c.ZPages.Port
	}
	return port, true
}

func (c *config) zipkinInterceptorEnabled() bool {
	if c == nil {
		return false
	}
	return c.Interceptors != nil && c.Interceptors.Zipkin != nil
}

func (c *config) zipkinInterceptorAddress() string {
	if c == nil || c.Interceptors == nil {
		return exporterparser.DefaultZipkinEndpointHostPort
	}
	inCfg := c.Interceptors
	if inCfg.Zipkin == nil || inCfg.Zipkin.Address == "" {
		return exporterparser.DefaultZipkinEndpointHostPort
	}
	return inCfg.Zipkin.Address
}

func parseOCAgentConfig(yamlBlob []byte) (*config, error) {
	var cfg config
	if err := yaml.Unmarshal(yamlBlob, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// The goal of this function is to catch logical errors such as
// if the Zipkin interceptor port conflicts with that of the exporter
// lest we'll have a self DOS because spans will be exported "out" from
// the exporter, yet be received from the interceptor, then sent back out
// and back in a never ending loop.
func (c *config) checkLogicalConflicts(blob []byte) error {
	var cfg struct {
		Exporters *struct {
			Zipkin *exporterparser.ZipkinConfig `yaml:"zipkin"`
		} `yaml:"exporters"`
	}
	if err := yaml.Unmarshal(blob, &cfg); err != nil {
		return err
	}

	if cfg.Exporters == nil || cfg.Exporters.Zipkin == nil || !c.zipkinInterceptorEnabled() {
		return nil
	}

	zc := cfg.Exporters.Zipkin

	zExporterAddr := zc.EndpointURL()
	zExporterURL, err := url.Parse(zExporterAddr)
	if err != nil {
		return fmt.Errorf("parsing ZipkinExporter address %q got error: %v", zExporterAddr, err)
	}

	zInterceptorHostPort := c.zipkinInterceptorAddress()

	zExporterHostPort := zExporterURL.Host
	if zInterceptorHostPort == zExporterHostPort {
		return fmt.Errorf("ZipkinExporter address (%q) is the same as the interceptor address (%q)",
			zExporterHostPort, zInterceptorHostPort)
	}
	zExpHost, zExpPort, _ := net.SplitHostPort(zExporterHostPort)
	zInterceptorHost, zInterceptorPort, _ := net.SplitHostPort(zExporterHostPort)
	if eqHosts(zExpHost, zInterceptorHost) && zExpPort == zInterceptorPort {
		return fmt.Errorf("ZipkinExporter address (%q) aka (%s on port %s)\nis the same as the interceptor address (%q) aka (%s on port %s)",
			zExporterHostPort, zExpHost, zExpPort, zInterceptorHostPort, zInterceptorHost, zInterceptorPort)
	}

	// Otherwise, now let's resolve the IPs and ensure that they aren't the same
	zExpIPAddr, _ := net.ResolveIPAddr("ip", zExpHost)
	zInterceptorIPAddr, _ := net.ResolveIPAddr("ip", zInterceptorHost)
	if zExpIPAddr != nil && zInterceptorIPAddr != nil && reflect.DeepEqual(zExpIPAddr, zInterceptorIPAddr) {
		return fmt.Errorf("ZipkinExporter address (%q) aka (%+v)\nis the same as the\ninterceptor address (%q) aka (%+v)",
			zExporterHostPort, zExpIPAddr, zInterceptorHostPort, zInterceptorIPAddr)
	}
	return nil
}

func eqHosts(host1, host2 string) bool {
	if host1 == host2 {
		return true
	}
	return eqLocalHost(host1) && eqLocalHost(host2)
}

func eqLocalHost(host string) bool {
	switch strings.ToLower(host) {
	case "", "localhost", "127.0.0.1":
		return true
	default:
		return false
	}
}

// exportersFromYAMLConfig parses the config yaml payload and returns the respective exporters
func exportersFromYAMLConfig(config []byte) (traceExporters []exporter.TraceExporter, doneFns []func() error) {
	parseFns := []struct {
		name string
		fn   func([]byte) ([]exporter.TraceExporter, []func() error, error)
	}{
		{name: "datadog", fn: exporterparser.DatadogTraceExportersFromYAML},
		{name: "stackdriver", fn: exporterparser.StackdriverTraceExportersFromYAML},
		{name: "zipkin", fn: exporterparser.ZipkinExportersFromYAML},
		{name: "jaeger", fn: exporterparser.JaegerExportersFromYAML},
	}

	for _, cfg := range parseFns {
		tes, tesDoneFns, err := cfg.fn(config)
		if err != nil {
			log.Fatalf("Failed to create config for %q: %v", cfg.name, err)
		}
		nonNilExporters := 0
		for _, te := range tes {
			if te != nil {
				traceExporters = append(traceExporters, te)
				nonNilExporters++
			}
		}

		if nonNilExporters > 0 {
			pluralization := "exporter"
			if nonNilExporters > 1 {
				pluralization = "exporters"
			}
			log.Printf("%q trace-%s enabled", cfg.name, pluralization)
		}
		for _, doneFn := range tesDoneFns {
			doneFns = append(doneFns, doneFn)
		}
	}
	return
}
