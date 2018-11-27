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
//  receivers:
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
	defaultOCReceiverAddress = ":55678"
	defaultZPagesPort        = 55679
)

type config struct {
	Receivers *receivers    `yaml:"receivers"`
	ZPages    *zPagesConfig `yaml:"zpages"`
}

type receivers struct {
	OpenCensus *receiverConfig `yaml:"opencensus"`
	Zipkin     *receiverConfig `yaml:"zipkin"`
	Jaeger     *receiverConfig `yaml:"jaeger"`
}

type receiverConfig struct {
	// The address to which the OpenCensus receiver will be bound and run on.
	Address             string `yaml:"address"`
	CollectorHTTPPort   int    `yaml:"collector_http_port"`
	CollectorThriftPort int    `yaml:"collector_thrift_port"`
}

type zPagesConfig struct {
	Disabled bool `yaml:"disabled"`
	Port     int  `yaml:"port"`
}

func (c *config) ocReceiverAddress() string {
	if c == nil || c.Receivers == nil {
		return defaultOCReceiverAddress
	}
	inCfg := c.Receivers
	if inCfg.OpenCensus == nil || inCfg.OpenCensus.Address == "" {
		return defaultOCReceiverAddress
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

func (c *config) zipkinReceiverEnabled() bool {
	if c == nil {
		return false
	}
	return c.Receivers != nil && c.Receivers.Zipkin != nil
}

func (c *config) jaegerReceiverEnabled() bool {
	if c == nil {
		return false
	}
	return c.Receivers != nil && c.Receivers.Jaeger != nil
}

func (c *config) zipkinReceiverAddress() string {
	if c == nil || c.Receivers == nil {
		return exporterparser.DefaultZipkinEndpointHostPort
	}
	inCfg := c.Receivers
	if inCfg.Zipkin == nil || inCfg.Zipkin.Address == "" {
		return exporterparser.DefaultZipkinEndpointHostPort
	}
	return inCfg.Zipkin.Address
}

func (c *config) jaegerReceiverPorts() (collectorPort, thriftPort int) {
	if c == nil || c.Receivers == nil {
		return 0, 0
	}
	rCfg := c.Receivers
	if rCfg.Jaeger == nil {
		return 0, 0
	}
	jc := rCfg.Jaeger
	return jc.CollectorHTTPPort, jc.CollectorThriftPort
}

func parseOCAgentConfig(yamlBlob []byte) (*config, error) {
	var cfg config
	if err := yaml.Unmarshal(yamlBlob, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// The goal of this function is to catch logical errors such as
// if the Zipkin receiver port conflicts with that of the exporter
// lest we'll have a self DOS because spans will be exported "out" from
// the exporter, yet be received from the receiver, then sent back out
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

	if cfg.Exporters == nil || cfg.Exporters.Zipkin == nil || !c.zipkinReceiverEnabled() {
		return nil
	}

	zc := cfg.Exporters.Zipkin

	zExporterAddr := zc.EndpointURL()
	zExporterURL, err := url.Parse(zExporterAddr)
	if err != nil {
		return fmt.Errorf("parsing ZipkinExporter address %q got error: %v", zExporterAddr, err)
	}

	zReceiverHostPort := c.zipkinReceiverAddress()

	zExporterHostPort := zExporterURL.Host
	if zReceiverHostPort == zExporterHostPort {
		return fmt.Errorf("ZipkinExporter address (%q) is the same as the receiver address (%q)",
			zExporterHostPort, zReceiverHostPort)
	}
	zExpHost, zExpPort, _ := net.SplitHostPort(zExporterHostPort)
	zReceiverHost, zReceiverPort, _ := net.SplitHostPort(zReceiverHostPort)
	if eqHosts(zExpHost, zReceiverHost) && zExpPort == zReceiverPort {
		return fmt.Errorf("ZipkinExporter address (%q) aka (%s on port %s)\nis the same as the receiver address (%q) aka (%s on port %s)",
			zExporterHostPort, zExpHost, zExpPort, zReceiverHostPort, zReceiverHost, zReceiverPort)
	}

	// Otherwise, now let's resolve the IPs and ensure that they aren't the same
	zExpIPAddr, _ := net.ResolveIPAddr("ip", zExpHost)
	zReceiverIPAddr, _ := net.ResolveIPAddr("ip", zReceiverHost)
	if zExpIPAddr != nil && zReceiverIPAddr != nil && reflect.DeepEqual(zExpIPAddr, zReceiverIPAddr) && zExpPort == zReceiverPort {
		return fmt.Errorf("ZipkinExporter address (%q) aka (%+v)\nis the same as the\nreceiver address (%q) aka (%+v)",
			zExporterHostPort, zExpIPAddr, zReceiverHostPort, zReceiverIPAddr)
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
		{name: "kafka", fn: exporterparser.KafkaExportersFromYAML},
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
