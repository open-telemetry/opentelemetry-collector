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

package config

import (
	"fmt"
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

// Config denotes the configuration for the various elements of an agent, that is:
// * Receivers
// * ZPages
// * Exporters
type Config struct {
	Receivers *Receivers    `yaml:"receivers"`
	ZPages    *ZPagesConfig `yaml:"zpages"`
	Exporters *Exporters    `yaml:"exporters"`
}

// Receivers denotes configurations for the various telemetry ingesters, such as:
// * Jaeger
// * OpenCensus
// * Zipkin
type Receivers struct {
	OpenCensus *ReceiverConfig `yaml:"opencensus"`
	Zipkin     *ReceiverConfig `yaml:"zipkin"`
	Jaeger     *ReceiverConfig `yaml:"jaeger"`
}

// ReceiverConfig is the per-receiver configuration that identifies attributes
// that a receiver's configuration can have such as:
// * Address
// * Various ports
type ReceiverConfig struct {
	// The address to which the OpenCensus receiver will be bound and run on.
	Address             string `yaml:"address"`
	CollectorHTTPPort   int    `yaml:"collector_http_port"`
	CollectorThriftPort int    `yaml:"collector_thrift_port"`

	// DisableTracing disables trace receiving and is only applicable to trace receivers.
	DisableTracing bool `yaml:"disable_tracing"`
	// DisableMetrics disables metrics receiving and is only applicable to metrics receivers.
	DisableMetrics bool `yaml:"disable_metrics"`
}

// Exporters denotes the configurations for the various backends
// that this service exports observability signals to.
type Exporters struct {
	Zipkin *exporterparser.ZipkinConfig `yaml:"zipkin"`
}

// ZPagesConfig denotes the configuration that zPages will be run with.
type ZPagesConfig struct {
	Disabled bool `yaml:"disabled"`
	Port     int  `yaml:"port"`
}

// OpenCensusReceiverAddress is a helper to safely retrieve the address
// that the OpenCensus receiver will be bound to.
// If Config is nil or the OpenCensus receiver's configuration is nil, it
// will return the default of ":55678"
func (c *Config) OpenCensusReceiverAddress() string {
	if c == nil || c.Receivers == nil {
		return defaultOCReceiverAddress
	}
	inCfg := c.Receivers
	if inCfg.OpenCensus == nil || inCfg.OpenCensus.Address == "" {
		return defaultOCReceiverAddress
	}
	return inCfg.OpenCensus.Address
}

// CanRunOpenCensusTraceReceiver returns true if the configuration
// permits running the OpenCensus Trace receiver.
func (c *Config) CanRunOpenCensusTraceReceiver() bool {
	return c != nil && c.Receivers != nil && !c.Receivers.OpenCensus.DisableTracing
}

// CanRunOpenCensusMetricsReceiver returns true if the configuration
// permits running the OpenCensus Metrics receiver.
func (c *Config) CanRunOpenCensusMetricsReceiver() bool {
	return c != nil && c.Receivers != nil && !c.Receivers.OpenCensus.DisableMetrics
}

// ZPagesDisabled returns true if zPages have not been enabled.
// It returns true if Config is nil or if ZPages are explicitly disabled.
func (c *Config) ZPagesDisabled() bool {
	if c == nil {
		return true
	}
	return c.ZPages != nil && c.ZPages.Disabled
}

// ZPagesPort tries to dereference the port on which zPages will be
// served.
// If zPages are disabled, it returns (-1, false)
// Else if no port is set, it returns the default  55679
func (c *Config) ZPagesPort() (int, bool) {
	if c.ZPagesDisabled() {
		return -1, false
	}
	port := defaultZPagesPort
	if c != nil && c.ZPages != nil && c.ZPages.Port > 0 {
		port = c.ZPages.Port
	}
	return port, true
}

// ZipkinReceiverEnabled returns true if Config is non-nil
// and if the Zipkin receiver configuration is also non-nil.
func (c *Config) ZipkinReceiverEnabled() bool {
	if c == nil {
		return false
	}
	return c.Receivers != nil && c.Receivers.Zipkin != nil
}

// JaegerReceiverEnabled returns true if Config is non-nil
// and if the Jaeger receiver configuration is also non-nil.
func (c *Config) JaegerReceiverEnabled() bool {
	if c == nil {
		return false
	}
	return c.Receivers != nil && c.Receivers.Jaeger != nil
}

// ZipkinReceiverAddress is a helper to safely retrieve the address
// that the Zipkin receiver will run on.
// If Config is nil or the Zipkin receiver's configuration is nil, it
// will return the default of "localhost:9411"
func (c *Config) ZipkinReceiverAddress() string {
	if c == nil || c.Receivers == nil {
		return exporterparser.DefaultZipkinEndpointHostPort
	}
	inCfg := c.Receivers
	if inCfg.Zipkin == nil || inCfg.Zipkin.Address == "" {
		return exporterparser.DefaultZipkinEndpointHostPort
	}
	return inCfg.Zipkin.Address
}

// JaegerReceiverPorts is a helper to safely retrieve the address
// that the Jaeger receiver will run on.
func (c *Config) JaegerReceiverPorts() (collectorPort, thriftPort int) {
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

// ParseOCAgentConfig unmarshals byte content in the YAML file format
// to  retrieve the configuration that will be used to run the OpenCensus agent.
func ParseOCAgentConfig(yamlBlob []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(yamlBlob, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// CheckLogicalConflicts serves to catch logical errors such as
// if the Zipkin receiver port conflicts with that of the exporter,
// lest we'll have a self DOS because spans will be exported "out" from
// the exporter, yet be received from the receiver, then sent back out
// and back in a never ending loop.
func (c *Config) CheckLogicalConflicts(blob []byte) error {
	var cfg Config
	if err := yaml.Unmarshal(blob, &cfg); err != nil {
		return err
	}

	if cfg.Exporters == nil || cfg.Exporters.Zipkin == nil || !c.ZipkinReceiverEnabled() {
		return nil
	}

	zc := cfg.Exporters.Zipkin

	zExporterAddr := zc.EndpointURL()
	zExporterURL, err := url.Parse(zExporterAddr)
	if err != nil {
		return fmt.Errorf("parsing ZipkinExporter address %q got error: %v", zExporterAddr, err)
	}

	zReceiverHostPort := c.ZipkinReceiverAddress()

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

// ExportersFromYAMLConfig parses the config yaml payload and returns the respective exporters
// from:
//  + datadog
//  + stackdriver
//  + zipkin
//  + jaeger
//  + kafka
//  + opencensus
func ExportersFromYAMLConfig(config []byte) (traceExporters []exporter.TraceExporter, metricsExporters []exporter.MetricsExporter, doneFns []func() error, err error) {
	parseFns := []struct {
		name string
		fn   func([]byte) ([]exporter.TraceExporter, []exporter.MetricsExporter, []func() error, error)
	}{
		{name: "datadog", fn: exporterparser.DatadogTraceExportersFromYAML},
		{name: "stackdriver", fn: exporterparser.StackdriverTraceExportersFromYAML},
		{name: "zipkin", fn: exporterparser.ZipkinExportersFromYAML},
		{name: "jaeger", fn: exporterparser.JaegerExportersFromYAML},
		{name: "kafka", fn: exporterparser.KafkaExportersFromYAML},
		{name: "opencensus", fn: exporterparser.OpenCensusTraceExportersFromYAML},
	}

	for _, cfg := range parseFns {
		tes, mes, tesDoneFns, terr := cfg.fn(config)
		if err != nil {
			err = fmt.Errorf("Failed to create config for %q: %v", cfg.name, terr)
			return
		}

		for _, te := range tes {
			if te != nil {
				traceExporters = append(traceExporters, te)
			}
		}

		for _, me := range mes {
			if me != nil {
				metricsExporters = append(metricsExporters, me)
			}
		}

		for _, doneFn := range tesDoneFns {
			doneFns = append(doneFns, doneFn)
		}
	}
	return
}
