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

// Program ocagent collects OpenCensus stats and traces
// to export to a configured backend.
package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"

	"go.opencensus.io/stats/view"
	"go.opencensus.io/zpages"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/census-instrumentation/opencensus-service/exporter"
	"github.com/census-instrumentation/opencensus-service/internal"
	"github.com/census-instrumentation/opencensus-service/internal/config"
	"github.com/census-instrumentation/opencensus-service/internal/config/viperutils"
	"github.com/census-instrumentation/opencensus-service/processor"
	"github.com/census-instrumentation/opencensus-service/receiver/jaegerreceiver"
	"github.com/census-instrumentation/opencensus-service/receiver/opencensusreceiver"
	"github.com/census-instrumentation/opencensus-service/receiver/prometheusreceiver"
	"github.com/census-instrumentation/opencensus-service/receiver/zipkinreceiver"
	"github.com/census-instrumentation/opencensus-service/receiver/zipkinreceiver/scribe"
)

var configYAMLFile string
var ocReceiverPort int

const zipkinRoute = "/api/v2/spans"

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runOCAgent() {
	yamlBlob, err := ioutil.ReadFile(configYAMLFile)
	if err != nil {
		log.Fatalf("Cannot read the YAML file %v error: %v", configYAMLFile, err)
	}
	agentConfig, err := config.ParseOCAgentConfig(yamlBlob)
	if err != nil {
		log.Fatalf("Failed to parse own configuration %v error: %v", configYAMLFile, err)
	}

	// Ensure that we check and catch any logical errors with the
	// configuration e.g. if an receiver shares the same address
	// as an exporter which would cause a self DOS and waste resources.
	if err := agentConfig.CheckLogicalConflicts(yamlBlob); err != nil {
		log.Fatalf("Configuration logical error: %v", err)
	}

	// TODO: don't hardcode info level logging
	conf := zap.NewProductionConfig()
	conf.Level.SetLevel(zapcore.InfoLevel)
	logger, err := conf.Build()
	if err != nil {
		log.Fatalf("Could not instantiate logger: %v", err)
	}

	// TODO(skaris): move the rest of the configs to use viper
	v, err := viperutils.ViperFromYAMLBytes([]byte(yamlBlob))
	if err != nil {
		log.Fatalf("Config: failed to create viper from YAML: %v", err)
	}
	traceExporters, metricsExporters, closeFns, err := config.ExportersFromViperConfig(logger, v)
	if err != nil {
		log.Fatalf("Config: failed to create exporters from YAML: %v", err)
	}

	commonSpanSink := exporter.MultiTraceExporters(traceExporters...)
	commonMetricsSink := exporter.MultiMetricsExporters(metricsExporters...)

	// Add other receivers here as they are implemented
	ocReceiverDoneFn, err := runOCReceiver(logger, agentConfig, commonSpanSink, commonMetricsSink)
	if err != nil {
		log.Fatal(err)
	}
	closeFns = append(closeFns, ocReceiverDoneFn)

	// If zPages are enabled, run them
	zPagesPort, zPagesEnabled := agentConfig.ZPagesPort()
	if zPagesEnabled {
		zCloseFn := runZPages(zPagesPort)
		closeFns = append(closeFns, zCloseFn)
	}

	// If the Zipkin receiver is enabled, then run it
	if agentConfig.ZipkinReceiverEnabled() {
		zipkinReceiverAddr := agentConfig.ZipkinReceiverAddress()
		zipkinReceiverDoneFn, err := runZipkinReceiver(zipkinReceiverAddr, commonSpanSink)
		if err != nil {
			log.Fatal(err)
		}
		closeFns = append(closeFns, zipkinReceiverDoneFn)
	}

	if agentConfig.ZipkinScribeReceiverEnabled() {
		zipkinScribeDoneFn, err := runZipkinScribeReceiver(agentConfig.ZipkinScribeConfig(), commonSpanSink)
		if err != nil {
			log.Fatal(err)
		}
		closeFns = append(closeFns, zipkinScribeDoneFn)
	}

	if agentConfig.JaegerReceiverEnabled() {
		collectorHTTPPort, collectorThriftPort := agentConfig.JaegerReceiverPorts()
		jaegerDoneFn, err := runJaegerReceiver(collectorThriftPort, collectorHTTPPort, commonSpanSink)
		if err != nil {
			log.Fatal(err)
		}
		closeFns = append(closeFns, jaegerDoneFn)
	}

	// If the Prometheus receiver is enabled, then run it.
	if agentConfig.PrometheusReceiverEnabled() {
		promDoneFn, err := runPrometheusReceiver(agentConfig.PrometheusConfiguration(), commonMetricsSink)
		if err != nil {
			log.Fatal(err)
		}
		closeFns = append(closeFns, promDoneFn)
	}

	// Always cleanup finally
	defer func() {
		for _, closeFn := range closeFns {
			if closeFn != nil {
				closeFn()
			}
		}
	}()

	signalsChan := make(chan os.Signal)
	signal.Notify(signalsChan, os.Interrupt)

	// Wait for the closing signal
	<-signalsChan
}

func runZPages(port int) func() error {
	// And enable zPages too
	zPagesMux := http.NewServeMux()
	zpages.Handle(zPagesMux, "/debug")

	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to bind to run zPages on %q: %v", addr, err)
	}

	srv := http.Server{Handler: zPagesMux}
	go func() {
		log.Printf("Running zPages at %q", addr)
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve zPages: %v", err)
		}
	}()

	return srv.Close
}

func runOCReceiver(logger *zap.Logger, acfg *config.Config, tdp processor.TraceDataProcessor, mdp processor.MetricsDataProcessor) (doneFn func() error, err error) {
	tlsCredsOption, hasTLSCreds, err := acfg.OpenCensusReceiverTLSCredentialsServerOption()
	if err != nil {
		return nil, fmt.Errorf("OpenCensus receiver TLS Credentials: %v", err)
	}
	addr := acfg.OpenCensusReceiverAddress()
	corsOrigins := acfg.OpenCensusReceiverCorsAllowedOrigins()
	ocr, err := opencensusreceiver.New(addr,
		tlsCredsOption,
		opencensusreceiver.WithCorsOrigins(corsOrigins))

	if err != nil {
		return nil, fmt.Errorf("Failed to create the OpenCensus receiver on address %q: error %v", addr, err)
	}
	if err := view.Register(internal.AllViews...); err != nil {
		return nil, fmt.Errorf("Failed to register internal.AllViews: %v", err)
	}

	// Temporarily disabling the grpc metrics since they do not provide good data at this moment,
	// See https://github.com/census-instrumentation/opencensus-service/issues/287
	// if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
	// 	return nil, fmt.Errorf("Failed to register ocgrpc.DefaultServerViews: %v", err)
	// }

	ctx := context.Background()

	switch {
	case acfg.CanRunOpenCensusTraceReceiver() && acfg.CanRunOpenCensusMetricsReceiver():
		if err := ocr.Start(ctx, tdp, mdp); err != nil {
			return nil, fmt.Errorf("Failed to start Trace and Metrics Receivers: %v", err)
		}
		log.Printf("Running OpenCensus Trace and Metrics receivers as a gRPC service at %q", addr)

	case acfg.CanRunOpenCensusTraceReceiver():
		if err := ocr.StartTraceReception(ctx, tdp); err != nil {
			return nil, fmt.Errorf("Failed to start TraceReceiver: %v", err)
		}
		log.Printf("Running OpenCensus Trace receiver as a gRPC service at %q", addr)

	case acfg.CanRunOpenCensusMetricsReceiver():
		if err := ocr.StartMetricsReception(ctx, mdp); err != nil {
			return nil, fmt.Errorf("Failed to start MetricsReceiver: %v", err)
		}
		log.Printf("Running OpenCensus Metrics receiver as a gRPC service at %q", addr)
	}

	if hasTLSCreds {
		tlsCreds := acfg.OpenCensusReceiverTLSServerCredentials()
		logger.Info("OpenCensus receiver with TLS Credentials",
			zap.String("cert_file", tlsCreds.CertFile),
			zap.String("key_file", tlsCreds.KeyFile))
	}

	doneFn = ocr.Stop
	return doneFn, nil
}

func runJaegerReceiver(collectorThriftPort, collectorHTTPPort int, next processor.TraceDataProcessor) (doneFn func() error, err error) {
	jtr, err := jaegerreceiver.New(context.Background(), &jaegerreceiver.Configuration{
		CollectorThriftPort: collectorThriftPort,
		CollectorHTTPPort:   collectorHTTPPort,

		// TODO: (@odeke-em, @pjanotti) send a change
		// to dynamically retrieve the Jaeger Agent's ports
		// and not use their defaults of 5778, 6831, 6832
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to create new Jaeger receiver: %v", err)
	}
	if err := jtr.StartTraceReception(context.Background(), next); err != nil {
		return nil, fmt.Errorf("Failed to start Jaeger receiver: %v", err)
	}
	doneFn = func() error {
		return jtr.StopTraceReception(context.Background())
	}
	log.Printf("Running Jaeger receiver with CollectorThriftPort %d CollectHTTPPort %d", collectorThriftPort, collectorHTTPPort)
	return doneFn, nil
}

func runZipkinReceiver(addr string, next processor.TraceDataProcessor) (doneFn func() error, err error) {
	zi, err := zipkinreceiver.New(addr)
	if err != nil {
		return nil, fmt.Errorf("Failed to create the Zipkin receiver: %v", err)
	}

	if err := zi.StartTraceReception(context.Background(), next); err != nil {
		return nil, fmt.Errorf("Cannot start Zipkin receiver with address %q: %v", addr, err)
	}
	doneFn = func() error {
		return zi.StopTraceReception(context.Background())
	}
	log.Printf("Running Zipkin receiver with address %q", addr)
	return doneFn, nil
}

func runZipkinScribeReceiver(config *config.ScribeReceiverConfig, next processor.TraceDataProcessor) (doneFn func() error, err error) {
	zs, err := scribe.NewReceiver(config.Address, config.Port, config.Category)
	if err != nil {
		return nil, fmt.Errorf("Failed to create the Zipkin Scribe receiver: %v", err)
	}

	if err := zs.StartTraceReception(context.Background(), next); err != nil {
		return nil, fmt.Errorf("Cannot start Zipkin Scribe receiver with %v: %v", config, err)
	}
	doneFn = func() error {
		return zs.StopTraceReception(context.Background())
	}
	log.Printf("Running Zipkin Scribe receiver with %+v", *config)
	return doneFn, nil
}

func runPrometheusReceiver(promConfig *prometheusreceiver.Configuration, next processor.MetricsDataProcessor) (doneFn func() error, err error) {
	pmr, err := prometheusreceiver.New(promConfig)
	if err != nil {
		return nil, err
	}
	if err := pmr.StartMetricsReception(context.Background(), next); err != nil {
		return nil, err
	}
	doneFn = func() error {
		return pmr.StopMetricsReception(context.Background())
	}
	log.Print("Running Prometheus receiver")
	return doneFn, nil
}
