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

// Package collector handles the command-line, configuration, and runs the OC collector.
package collector

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/census-instrumentation/opencensus-service/cmd/occollector/app/builder"
	"github.com/census-instrumentation/opencensus-service/exporter"
	"github.com/census-instrumentation/opencensus-service/exporter/exporterparser"
	"github.com/census-instrumentation/opencensus-service/internal/collector/jaeger"
	"github.com/census-instrumentation/opencensus-service/internal/collector/opencensus"
	"github.com/census-instrumentation/opencensus-service/internal/collector/processor"
	"github.com/census-instrumentation/opencensus-service/internal/collector/zipkin"
)

const (
	configCfg          = "config"
	logLevelCfg        = "log-level"
	jaegerReceiverFlg  = "receive-jaeger"
	ocReceiverFlg      = "receive-oc-trace"
	zipkinReceiverFlg  = "receive-zipkin"
	debugProcessorFlg  = "debug-processor"
	queuedProcessorFlg = "add-queued-processor" // TODO: (@pjanotti) this is temporary flag until it can be read from config.
)

var (
	config string

	v = viper.New()

	logger *zap.Logger

	rootCmd = &cobra.Command{
		Use:  "occollector",
		Long: "OpenCensus Collector",
		Run: func(cmd *cobra.Command, args []string) {
			if file := v.GetString(configCfg); file != "" {
				v.SetConfigFile(file)
				err := v.ReadInConfig()
				if err != nil {
					log.Fatalf("Error loading config file %q: %v", file, err)
					return
				}
			}

			execute()
		},
	}
)

func init() {
	rootCmd.PersistentFlags().String(logLevelCfg, "INFO", "Output level of logs (TRACE, DEBUG, INFO, WARN, ERROR, FATAL)")
	v.BindPFlag(logLevelCfg, rootCmd.PersistentFlags().Lookup(logLevelCfg))

	// local flags
	rootCmd.Flags().StringVar(&config, configCfg, "", "Path to the config file")
	rootCmd.Flags().Bool(jaegerReceiverFlg, false,
		fmt.Sprintf("Flag to run the Jaeger receiver (i.e.: Jaeger Collector), default settings: %+v", *builder.NewDefaultJaegerReceiverCfg()))
	rootCmd.Flags().Bool(ocReceiverFlg, false,
		fmt.Sprintf("Flag to run the OpenCensus trace receiver, default settings: %+v", *builder.NewDefaultOpenCensusReceiverCfg()))
	rootCmd.Flags().Bool(zipkinReceiverFlg, false,
		fmt.Sprintf("Flag to run the Zipkin receiver, default settings: %+v", *builder.NewDefaultZipkinReceiverCfg()))
	rootCmd.Flags().Bool(debugProcessorFlg, false, "Flag to add a debug processor (combine with log level DEBUG to log incoming spans)")
	rootCmd.Flags().Bool(queuedProcessorFlg, false, "Flag to wrap one processor with the queued processor (flag will be remove soon, dev helper)")

	// TODO: (@pjanotti) add builder options as flags, before calls bellow. Likely it will require code re-org.

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	v.BindPFlags(rootCmd.Flags())
}

func newLogger() (*zap.Logger, error) {
	var level zapcore.Level
	err := (&level).UnmarshalText([]byte(v.GetString(logLevelCfg)))
	if err != nil {
		return nil, err
	}
	conf := zap.NewProductionConfig()
	conf.Level.SetLevel(level)
	return conf.Build()
}

func execute() {
	var signalsChannel = make(chan os.Signal)
	signal.Notify(signalsChannel, os.Interrupt, syscall.SIGTERM)

	var err error
	logger, err = newLogger()
	if err != nil {
		log.Fatalf("Failed to get logger: %v", err)
	}

	logger.Info("Starting...")

	// Build pipeline from its end: 1st exporters, the OC-proto queue processor, and
	// finally the receivers.

	var closeFns []func()
	var spanProcessors []processor.SpanProcessor
	exportersCloseFns, exporters := createExporters()
	closeFns = append(closeFns, exportersCloseFns...)
	if len(exporters) > 0 {
		// Exporters need an extra hop from OC-proto to span data: to workaround that for now
		// we will use a special processor that transforms the data to a format that they can consume.
		// TODO: (@pjanotti) we should avoid this step in the long run, its an extra hop just to re-use
		// the exporters: this can lose node information and it is not ideal for performance and delegates
		// the retry/buffering to the exporters (that are designed to run within the tracing process).
		spanProcessors = append(spanProcessors, processor.NewTraceExporterProcessor(exporters...))
	}

	if v.GetBool(debugProcessorFlg) {
		spanProcessors = append(spanProcessors, processor.NewNoopSpanProcessor(logger))
	}

	if len(spanProcessors) == 0 {
		logger.Warn("Nothing to do: no processor was enabled. Shutting down.")
		os.Exit(1)
	}

	// TODO: (@pjanotti) construct queued processor from config options, for now just to exercise it, wrap one around
	// the first span processor available.
	if v.GetBool(queuedProcessorFlg) {
		spanProcessors[0] = processor.NewQueuedSpanProcessor(spanProcessors[0])
	}

	// Wraps processors in a single one to be connected to all enabled receivers.
	spanProcessor := processor.NewMultiSpanProcessor(spanProcessors...)

	receiversCloseFns := createReceivers(spanProcessor)
	closeFns = append(closeFns, receiversCloseFns...)

	logger.Info("Collector is up and running.")

	<-signalsChannel
	logger.Info("Starting shutdown...")

	// TODO: orderly shutdown: first receivers, then flushing pipelines giving
	// senders a chance to send all their data. This may take time, the allowed
	// time should be part of configuration.
	for i := len(closeFns) - 1; i > 0; i-- {
		closeFns[i]()
	}

	logger.Info("Shutdown complete.")
}

func createExporters() (doneFns []func(), traceExporters []exporter.TraceExporter) {
	// TODO: (@pjanotti) this is slightly modified from agent but in the end duplication, need to consolidate style and visibility.
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

	if config == "" {
		logger.Info("No config file, exporters can be only configured via the file.")
		return
	}

	cfgBlob, err := ioutil.ReadFile(config)
	if err != nil {
		logger.Fatal("Cannot read config file for exporters", zap.Error(err))
	}

	for _, cfg := range parseFns {
		tes, tesDoneFns, err := cfg.fn(cfgBlob)
		if err != nil {
			logger.Fatal("Failed to create config for exporter", zap.String("exporter", cfg.name), zap.Error(err))
		}

		wasEnabled := false
		for _, te := range tes {
			if te != nil {
				wasEnabled = true
				traceExporters = append(traceExporters, te)
			}
		}

		for _, tesDoneFn := range tesDoneFns {
			if tesDoneFn != nil {
				wrapperFn := func() {
					if err := tesDoneFn(); err != nil {
						logger.Warn("Error when closing exporter", zap.String("exporter", cfg.name), zap.Error(err))
					}
				}
				doneFns = append(doneFns, wrapperFn)
			}
		}

		if wasEnabled {
			logger.Info("Exporter enabled", zap.String("exporter", cfg.name))
		}
	}

	return doneFns, traceExporters
}

func createReceivers(spanProcessor processor.SpanProcessor) (closeFns []func()) {
	var someReceiverEnabled bool
	receivers := []struct {
		name    string
		runFn   func(*zap.Logger, *viper.Viper, processor.SpanProcessor) (func(), error)
		enabled bool
	}{
		{"Jaeger", jaegerreceiver.Run, builder.JaegerReceiverEnabled(v, jaegerReceiverFlg)},
		{"OpenCensus", ocreceiver.Run, builder.OpenCensusReceiverEnabled(v, ocReceiverFlg)},
		{"Zipkin", zipkinreceiver.Run, builder.ZipkinReceiverEnabled(v, zipkinReceiverFlg)},
	}

	for _, receiver := range receivers {
		if receiver.enabled {
			closeSrv, err := receiver.runFn(logger, v, spanProcessor)
			if err != nil {
				// TODO: (@pjanotti) better shutdown, for now just try to stop any started receiver before terminating.
				for _, closeFn := range closeFns {
					closeFn()
				}
				logger.Fatal("Cannot run receiver for "+receiver.name, zap.Error(err))
			}
			closeFns = append(closeFns, closeSrv)
			someReceiverEnabled = true
		}
	}

	if !someReceiverEnabled {
		logger.Warn("Nothing to do: no receiver was enabled. Shutting down.")
		os.Exit(1)
	}

	return closeFns
}

// Execute the application according to the command and configuration given
// by the user.
func Execute() error {
	return rootCmd.Execute()
}
