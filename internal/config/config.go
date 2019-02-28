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

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	yaml "gopkg.in/yaml.v2"

	"github.com/census-instrumentation/opencensus-service/exporter/awsexporter"
	"github.com/census-instrumentation/opencensus-service/exporter/datadogexporter"
	"github.com/census-instrumentation/opencensus-service/exporter/honeycombexporter"
	"github.com/census-instrumentation/opencensus-service/exporter/jaegerexporter"
	"github.com/census-instrumentation/opencensus-service/exporter/kafkaexporter"
	"github.com/census-instrumentation/opencensus-service/exporter/opencensusexporter"
	"github.com/census-instrumentation/opencensus-service/exporter/prometheusexporter"
	"github.com/census-instrumentation/opencensus-service/exporter/stackdriverexporter"
	"github.com/census-instrumentation/opencensus-service/exporter/zipkinexporter"
	"github.com/census-instrumentation/opencensus-service/processor"
	"github.com/census-instrumentation/opencensus-service/receiver/opencensusreceiver"
	"github.com/census-instrumentation/opencensus-service/receiver/prometheusreceiver"
)

// We expect the configuration.yaml file to look like this:
//
//  receivers:
//      opencensus:
//          port: <port>
//
//      zipkin:
//          reporter: <address>
//
//      prometheus:
//          config:
//             scrape_configs:
//               - job_name: 'foo_service"
//                 scrape_interval: 5s
//                 static_configs:
//                   - targets: ['localhost:8889']
//          buffer_count: 10
//          buffer_period: 5s
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

var defaultOCReceiverCorsAllowedOrigins = []string{}

var defaultScribeConfiguration = &ScribeReceiverConfig{
	Port:     9410,
	Category: "zipkin",
}

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
// * Jaeger (traces)
// * OpenCensus (metrics and traces)
// * Prometheus (metrics)
// * Zipkin (traces)
type Receivers struct {
	OpenCensus *ReceiverConfig       `yaml:"opencensus"`
	Zipkin     *ReceiverConfig       `yaml:"zipkin"`
	Jaeger     *ReceiverConfig       `yaml:"jaeger"`
	Scribe     *ScribeReceiverConfig `yaml:"zipkin-scribe"`

	// Prometheus contains the Prometheus configurations.
	// Such as:
	//      scrape_configs:
	//          - job_name: "agent"
	//              scrape_interval: 5s
	//
	//          static_configs:
	//              - targets: ['localhost:9988']
	Prometheus *prometheusreceiver.Configuration `yaml:"prometheus"`
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

	// The allowed CORS origins for HTTP/JSON requests the grpc-gateway adapter
	// for the OpenCensus receiver. See github.com/rs/cors
	// An empty list means that CORS is not enabled at all. A wildcard (*) can be
	// used to match any origin or one or more characters of an origin.
	CorsAllowedOrigins []string `yaml:"cors_allowed_origins"`

	// DisableTracing disables trace receiving and is only applicable to trace receivers.
	DisableTracing bool `yaml:"disable_tracing"`
	// DisableMetrics disables metrics receiving and is only applicable to metrics receivers.
	DisableMetrics bool `yaml:"disable_metrics"`

	// TLSCredentials is a (cert_file, key_file) configuration.
	TLSCredentials *TLSCredentials `yaml:"tls_credentials"`
}

// ScribeReceiverConfig carries the settings for the Zipkin Scribe receiver.
type ScribeReceiverConfig struct {
	// Address is an IP address or a name that can be resolved to a local address.
	//
	// It can use a name, but this is not recommended, because it will create
	// a listener for at most one of the host's IP addresses.
	//
	// The default value bind to all available interfaces on the local computer.
	Address string `yaml:"address" mapstructure:"address"`
	Port    uint16 `yaml:"port" mapstructure:"port"`
	// Category is the string that will be used to identify the scribe log messages
	// that contain Zipkin spans.
	Category string `yaml:"category" mapstructure:"category"`
}

// Exporters denotes the configurations for the various backends
// that this service exports observability signals to.
type Exporters struct {
	Zipkin *zipkinexporter.ZipkinConfig `yaml:"zipkin"`
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

// OpenCensusReceiverCorsAllowedOrigins is a helper to safely retrieve the list
// of allowed CORS origins for HTTP/JSON requests to the grpc-gateway adapter.
func (c *Config) OpenCensusReceiverCorsAllowedOrigins() []string {
	if c == nil || c.Receivers == nil {
		return defaultOCReceiverCorsAllowedOrigins
	}
	inCfg := c.Receivers
	if inCfg.OpenCensus == nil {
		return defaultOCReceiverCorsAllowedOrigins
	}
	return inCfg.OpenCensus.CorsAllowedOrigins
}

// CanRunOpenCensusTraceReceiver returns true if the configuration
// permits running the OpenCensus Trace receiver.
func (c *Config) CanRunOpenCensusTraceReceiver() bool {
	return c.openCensusReceiverEnabled() && !c.Receivers.OpenCensus.DisableTracing
}

// CanRunOpenCensusMetricsReceiver returns true if the configuration
// permits running the OpenCensus Metrics receiver.
func (c *Config) CanRunOpenCensusMetricsReceiver() bool {
	return c.openCensusReceiverEnabled() && !c.Receivers.OpenCensus.DisableMetrics
}

// openCensusReceiverEnabled returns true if both:
//    Config.Receivers and Config.Receivers.OpenCensus
// are non-nil.
func (c *Config) openCensusReceiverEnabled() bool {
	return c != nil && c.Receivers != nil &&
		c.Receivers.OpenCensus != nil
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

// ZipkinScribeReceiverEnabled returns true if Config is non-nil
// and if the Scribe receiver configuration is also non-nil.
func (c *Config) ZipkinScribeReceiverEnabled() bool {
	if c == nil {
		return false
	}
	return c.Receivers != nil && c.Receivers.Scribe != nil
}

// JaegerReceiverEnabled returns true if Config is non-nil
// and if the Jaeger receiver configuration is also non-nil.
func (c *Config) JaegerReceiverEnabled() bool {
	if c == nil {
		return false
	}
	return c.Receivers != nil && c.Receivers.Jaeger != nil
}

// PrometheusReceiverEnabled returns true if Config is non-nil
// and if the Jaeger receiver configuration is also non-nil.
func (c *Config) PrometheusReceiverEnabled() bool {
	if c == nil {
		return false
	}
	return c.Receivers != nil && c.Receivers.Prometheus != nil
}

// PrometheusConfiguration deferences and returns the Prometheus configuration
// if non-nil.
func (c *Config) PrometheusConfiguration() *prometheusreceiver.Configuration {
	if c == nil || c.Receivers == nil {
		return nil
	}
	return c.Receivers.Prometheus
}

// ZipkinReceiverAddress is a helper to safely retrieve the address
// that the Zipkin receiver will run on.
// If Config is nil or the Zipkin receiver's configuration is nil, it
// will return the default of "localhost:9411"
func (c *Config) ZipkinReceiverAddress() string {
	if c == nil || c.Receivers == nil {
		return zipkinexporter.DefaultZipkinEndpointHostPort
	}
	inCfg := c.Receivers
	if inCfg.Zipkin == nil || inCfg.Zipkin.Address == "" {
		return zipkinexporter.DefaultZipkinEndpointHostPort
	}
	return inCfg.Zipkin.Address
}

// ZipkinScribeConfig is a helper to safely retrieve the Zipkin Scribe
// configuration.
func (c *Config) ZipkinScribeConfig() *ScribeReceiverConfig {
	if c == nil || c.Receivers == nil || c.Receivers.Scribe == nil {
		return defaultScribeConfiguration
	}
	cfg := c.Receivers.Scribe
	if cfg.Port == 0 {
		cfg.Port = defaultScribeConfiguration.Port
	}
	if cfg.Category == "" {
		cfg.Category = defaultScribeConfiguration.Category
	}
	return cfg
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

// HasTLSCredentials returns true if TLSCredentials is non-nil
func (rCfg *ReceiverConfig) HasTLSCredentials() bool {
	return rCfg != nil && rCfg.TLSCredentials != nil && rCfg.TLSCredentials.nonEmpty()
}

// OpenCensusReceiverTLSServerCredentials retrieves the TLS credentials
// from this Config's OpenCensus receiver if any.
func (c *Config) OpenCensusReceiverTLSServerCredentials() *TLSCredentials {
	if !c.openCensusReceiverEnabled() {
		return nil
	}

	ocrConfig := c.Receivers.OpenCensus
	if !ocrConfig.HasTLSCredentials() {
		return nil
	}
	return ocrConfig.TLSCredentials
}

// ToOpenCensusReceiverServerOption checks if the TLS credentials
// in the form of a certificate file and a key file. If they aren't,
// it will return opencensusreceiver.WithNoopOption() and a nil error.
// Otherwise, it will try to retrieve gRPC transport credentials from the file combinations,
// and create a option, along with any errors encountered while retrieving the credentials.
func (tlsCreds *TLSCredentials) ToOpenCensusReceiverServerOption() (opt opencensusreceiver.Option, ok bool, err error) {
	if tlsCreds == nil {
		return opencensusreceiver.WithNoopOption(), false, nil
	}

	transportCreds, err := credentials.NewServerTLSFromFile(tlsCreds.CertFile, tlsCreds.KeyFile)
	if err != nil {
		return nil, false, err
	}
	gRPCCredsOpt := grpc.Creds(transportCreds)
	return opencensusreceiver.WithGRPCServerOptions(gRPCCredsOpt), true, nil
}

// OpenCensusReceiverTLSCredentialsServerOption checks if the OpenCensus receiver's Configuration
// has TLS credentials in the form of a certificate file and a key file. If it doesn't
// have any, it will return opencensusreceiver.WithNoopOption() and a nil error.
// Otherwise, it will try to retrieve gRPC transport credentials from the file combinations,
// and create a option, along with any errors encountered while retrieving the credentials.
func (c *Config) OpenCensusReceiverTLSCredentialsServerOption() (opt opencensusreceiver.Option, ok bool, err error) {
	tlsCreds := c.OpenCensusReceiverTLSServerCredentials()
	return tlsCreds.ToOpenCensusReceiverServerOption()
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

// ExportersFromViperConfig uses the viper configuration payload to returns the respective exporters
// from:
//  + datadog
//  + stackdriver
//  + zipkin
//  + jaeger
//  + kafka
//  + opencensus
//  + prometheus
//  + aws-xray
//  + honeycomb
func ExportersFromViperConfig(logger *zap.Logger, v *viper.Viper) ([]processor.TraceDataProcessor, []processor.MetricsDataProcessor, []func() error, error) {
	parseFns := []struct {
		name string
		fn   func(*viper.Viper) ([]processor.TraceDataProcessor, []processor.MetricsDataProcessor, []func() error, error)
	}{
		{name: "datadog", fn: datadogexporter.DatadogTraceExportersFromViper},
		{name: "stackdriver", fn: stackdriverexporter.StackdriverTraceExportersFromViper},
		{name: "zipkin", fn: zipkinexporter.ZipkinExportersFromViper},
		{name: "jaeger", fn: jaegerexporter.JaegerExportersFromViper},
		{name: "kafka", fn: kafkaexporter.KafkaExportersFromViper},
		{name: "opencensus", fn: opencensusexporter.OpenCensusTraceExportersFromViper},
		{name: "prometheus", fn: prometheusexporter.PrometheusExportersFromViper},
		{name: "aws-xray", fn: awsexporter.AWSXRayTraceExportersFromViper},
		{name: "honeycomb", fn: honeycombexporter.HoneycombTraceExportersFromViper},
	}

	var traceExporters []processor.TraceDataProcessor
	var metricsExporters []processor.MetricsDataProcessor
	var doneFns []func() error
	exportersViper := v.Sub("exporters")
	if exportersViper == nil {
		return nil, nil, nil, nil
	}
	for _, cfg := range parseFns {
		tes, mes, tesDoneFns, err := cfg.fn(exportersViper)
		if err != nil {
			err = fmt.Errorf("Failed to create config for %q: %v", cfg.name, err)
			return nil, nil, nil, err
		}

		for _, te := range tes {
			if te != nil {
				traceExporters = append(traceExporters, te)
				logger.Info("Trace Exporter enabled", zap.String("exporter", cfg.name))
			}
		}

		for _, me := range mes {
			if me != nil {
				metricsExporters = append(metricsExporters, me)
				logger.Info("Metrics Exporter enabled", zap.String("exporter", cfg.name))
			}
		}

		for _, doneFn := range tesDoneFns {
			if doneFn != nil {
				doneFns = append(doneFns, doneFn)
			}
		}
	}
	return traceExporters, metricsExporters, doneFns, nil
}
