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
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/zpages"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/census-instrumentation/opencensus-service/consumer"
	"github.com/census-instrumentation/opencensus-service/internal/config"
	"github.com/census-instrumentation/opencensus-service/internal/config/viperutils"
	"github.com/census-instrumentation/opencensus-service/internal/pprofserver"
	"github.com/census-instrumentation/opencensus-service/internal/version"
	"github.com/census-instrumentation/opencensus-service/observability"
	"github.com/census-instrumentation/opencensus-service/processor/multiconsumer"
	"github.com/census-instrumentation/opencensus-service/receiver/jaegerreceiver"
	"github.com/census-instrumentation/opencensus-service/receiver/opencensusreceiver"
	"github.com/census-instrumentation/opencensus-service/receiver/prometheusreceiver"
	"github.com/census-instrumentation/opencensus-service/receiver/zipkinreceiver"
	"github.com/census-instrumentation/opencensus-service/receiver/zipkinreceiver/zipkinscribereceiver"
)

var rootCmd = &cobra.Command{
	Use:   "ocagent",
	Short: "ocagent runs the OpenCensus service",
	Run: func(cmd *cobra.Command, args []string) {
		runOCAgent()
	},
}

var viperCfg = viper.New()

var configYAMLFile string

func init() {
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version information for ocagent",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(version.Info())
		},
	}
	rootCmd.AddCommand(versionCmd)
	rootCmd.PersistentFlags().StringVarP(&configYAMLFile, "config", "c", "config.yaml", "The YAML file with the configurations for the agent and various exporters")

	viperutils.AddFlags(viperCfg, rootCmd, pprofserver.AddFlags)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runOCAgent() {
	viperCfg.SetConfigFile(configYAMLFile)
	err := viperCfg.ReadInConfig()
	if err != nil {
		log.Fatalf("Cannot read the YAML file %v error: %v", configYAMLFile, err)
	}

	var agentConfig config.Config
	err = viperCfg.Unmarshal(&agentConfig)
	if err != nil {
		log.Fatalf("Error unmarshalling yaml config file %v: %v", configYAMLFile, err)
	}

	// Ensure that we check and catch any logical errors with the
	// configuration e.g. if an receiver shares the same address
	// as an exporter which would cause a self DOS and waste resources.
	if err := agentConfig.CheckLogicalConflicts(); err != nil {
		log.Fatalf("Configuration logical error: %v", err)
	}

	// TODO: don't hardcode info level logging
	conf := zap.NewProductionConfig()
	conf.Level.SetLevel(zapcore.InfoLevel)
	logger, err := conf.Build()
	if err != nil {
		log.Fatalf("Could not instantiate logger: %v", err)
	}

	var asyncErrorChan = make(chan error)
	err = pprofserver.SetupFromViper(asyncErrorChan, viperCfg, logger)
	if err != nil {
		log.Fatalf("Failed to start net/http/pprof: %v", err)
	}

	traceExporters, metricsExporters, closeFns, err := config.ExportersFromViperConfig(logger, viperCfg)
	if err != nil {
		log.Fatalf("Config: failed to create exporters from YAML: %v", err)
	}

	commonSpanSink := multiconsumer.NewTraceProcessor(traceExporters)
	commonMetricsSink := multiconsumer.NewMetricsProcessor(metricsExporters)

	// Add other receivers here as they are implemented
	ocReceiverDoneFn, err := runOCReceiver(logger, &agentConfig, commonSpanSink, commonMetricsSink, asyncErrorChan)
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

	// TODO: Generalize the startup of these receivers when unifying them w/ collector
	// If the Zipkin receiver is enabled, then run it
	if agentConfig.ZipkinReceiverEnabled() {
		zipkinReceiverAddr := agentConfig.ZipkinReceiverAddress()
		zipkinReceiverDoneFn, err := runZipkinReceiver(zipkinReceiverAddr, commonSpanSink, asyncErrorChan)
		if err != nil {
			log.Fatal(err)
		}
		closeFns = append(closeFns, zipkinReceiverDoneFn)
	}

	if agentConfig.ZipkinScribeReceiverEnabled() {
		zipkinScribeDoneFn, err := runZipkinScribeReceiver(agentConfig.ZipkinScribeConfig(), commonSpanSink, asyncErrorChan)
		if err != nil {
			log.Fatal(err)
		}
		closeFns = append(closeFns, zipkinScribeDoneFn)
	}

	if agentConfig.JaegerReceiverEnabled() {
		collectorHTTPPort, collectorThriftPort := agentConfig.JaegerReceiverPorts()
		jaegerDoneFn, err := runJaegerReceiver(collectorThriftPort, collectorHTTPPort, commonSpanSink, asyncErrorChan)
		if err != nil {
			log.Fatal(err)
		}
		closeFns = append(closeFns, jaegerDoneFn)
	}

	// If the Prometheus receiver is enabled, then run it.
	if agentConfig.PrometheusReceiverEnabled() {
		promDoneFn, err := runPrometheusReceiver(viperCfg, commonMetricsSink, asyncErrorChan)
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

	signalsChan := make(chan os.Signal, 1)
	signal.Notify(signalsChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err = <-asyncErrorChan:
		log.Fatalf("Asynchronous error %q, terminating process", err)
	case s := <-signalsChan:
		log.Printf("Received %q signal from OS, terminating process", s)
	}
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

func runOCReceiver(logger *zap.Logger, acfg *config.Config, tc consumer.TraceConsumer, mc consumer.MetricsConsumer, asyncErrorChan chan<- error) (doneFn func() error, err error) {
	tlsCredsOption, hasTLSCreds, err := acfg.OpenCensusReceiverTLSCredentialsServerOption()
	if err != nil {
		return nil, fmt.Errorf("OpenCensus receiver TLS Credentials: %v", err)
	}
	addr := acfg.OpenCensusReceiverAddress()
	corsOrigins := acfg.OpenCensusReceiverCorsAllowedOrigins()
	ocr, err := opencensusreceiver.New(addr,
		tc,
		mc,
		tlsCredsOption,
		opencensusreceiver.WithCorsOrigins(corsOrigins))

	if err != nil {
		return nil, fmt.Errorf("failed to create the OpenCensus receiver on address %q: error %v", addr, err)
	}
	if err := view.Register(observability.AllViews...); err != nil {
		return nil, fmt.Errorf("failed to register internal.AllViews: %v", err)
	}

	// Temporarily disabling the grpc metrics since they do not provide good data at this moment,
	// See https://github.com/census-instrumentation/opencensus-service/issues/287
	// if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
	// 	return nil, fmt.Errorf("Failed to register ocgrpc.DefaultServerViews: %v", err)
	// }

	ctx := context.Background()

	switch {
	case acfg.CanRunOpenCensusTraceReceiver() && acfg.CanRunOpenCensusMetricsReceiver():
		if err := ocr.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start Trace and Metrics Receivers: %v", err)
		}
		log.Printf("Running OpenCensus Trace and Metrics receivers as a gRPC service at %q", addr)

	case acfg.CanRunOpenCensusTraceReceiver():
		if err := ocr.StartTraceReception(ctx, asyncErrorChan); err != nil {
			return nil, fmt.Errorf("failed to start TraceReceiver: %v", err)
		}
		log.Printf("Running OpenCensus Trace receiver as a gRPC service at %q", addr)

	case acfg.CanRunOpenCensusMetricsReceiver():
		if err := ocr.StartMetricsReception(ctx, asyncErrorChan); err != nil {
			return nil, fmt.Errorf("failed to start MetricsReceiver: %v", err)
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

func runJaegerReceiver(collectorThriftPort, collectorHTTPPort int, next consumer.TraceConsumer, asyncErrorChan chan<- error) (doneFn func() error, err error) {
	config := &jaegerreceiver.Configuration{
		CollectorThriftPort: collectorThriftPort,
		CollectorHTTPPort:   collectorHTTPPort,

		// TODO: (@odeke-em, @pjanotti) send a change
		// to dynamically retrieve the Jaeger Agent's ports
		// and not use their defaults of 5778, 6831, 6832
	}
	jtr, err := jaegerreceiver.New(context.Background(), config, next)
	if err != nil {
		return nil, fmt.Errorf("failed to create new Jaeger receiver: %v", err)
	}
	if err := jtr.StartTraceReception(context.Background(), asyncErrorChan); err != nil {
		return nil, fmt.Errorf("failed to start Jaeger receiver: %v", err)
	}
	doneFn = func() error {
		return jtr.StopTraceReception(context.Background())
	}
	log.Printf("Running Jaeger receiver with CollectorThriftPort %d CollectHTTPPort %d", collectorThriftPort, collectorHTTPPort)
	return doneFn, nil
}

func runZipkinReceiver(addr string, next consumer.TraceConsumer, asyncErrorChan chan<- error) (doneFn func() error, err error) {
	zi, err := zipkinreceiver.New(addr, next)
	if err != nil {
		return nil, fmt.Errorf("failed to create the Zipkin receiver: %v", err)
	}

	if err := zi.StartTraceReception(context.Background(), asyncErrorChan); err != nil {
		return nil, fmt.Errorf("cannot start Zipkin receiver with address %q: %v", addr, err)
	}
	doneFn = func() error {
		return zi.StopTraceReception(context.Background())
	}
	log.Printf("Running Zipkin receiver with address %q", addr)
	return doneFn, nil
}

func runZipkinScribeReceiver(config *config.ScribeReceiverConfig, next consumer.TraceConsumer, asyncErrorChan chan<- error) (doneFn func() error, err error) {
	zs, err := zipkinscribereceiver.NewReceiver(config.Address, config.Port, config.Category, next)
	if err != nil {
		return nil, fmt.Errorf("failed to create the Zipkin Scribe receiver: %v", err)
	}

	if err := zs.StartTraceReception(context.Background(), asyncErrorChan); err != nil {
		return nil, fmt.Errorf("cannot start Zipkin Scribe receiver with %v: %v", config, err)
	}
	doneFn = func() error {
		return zs.StopTraceReception(context.Background())
	}
	log.Printf("Running Zipkin Scribe receiver with %+v", *config)
	return doneFn, nil
}

func runPrometheusReceiver(v *viper.Viper, next consumer.MetricsConsumer, asyncErrorChan chan<- error) (doneFn func() error, err error) {
	pmr, err := prometheusreceiver.New(v.Sub("receivers.prometheus"), next)
	if err != nil {
		return nil, err
	}
	if err := pmr.StartMetricsReception(context.Background(), asyncErrorChan); err != nil {
		return nil, err
	}
	doneFn = func() error {
		return pmr.StopMetricsReception(context.Background())
	}
	log.Print("Running Prometheus receiver")
	return doneFn, nil
}
