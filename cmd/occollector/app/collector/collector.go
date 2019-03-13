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
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/jaegertracing/jaeger/pkg/healthcheck"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/census-instrumentation/opencensus-service/cmd/occollector/app/builder"
	"github.com/census-instrumentation/opencensus-service/consumer"
	"github.com/census-instrumentation/opencensus-service/internal/config/viperutils"
	"github.com/census-instrumentation/opencensus-service/internal/pprofserver"
	"github.com/census-instrumentation/opencensus-service/receiver"
)

var (
	// App represents the collector application in its entirety
	App = newApp()
)

// Application represents a collector application
type Application struct {
	v           *viper.Viper
	logger      *zap.Logger
	healthCheck *healthcheck.HealthCheck
	processor   consumer.TraceConsumer
	receivers   []receiver.TraceReceiver
}

func newApp() *Application {
	return &Application{
		v: viper.New(),
	}
}

func (app *Application) init() {
	var err error
	if file := builder.GetConfigFile(app.v); file != "" {
		app.v.SetConfigFile(file)
		err := app.v.ReadInConfig()
		if err != nil {
			log.Fatalf("Error loading config file %q: %v", file, err)
			return
		}
	}
	app.logger, err = newLogger(app.v)
	if err != nil {
		log.Fatalf("Failed to get logger: %v", err)
	}
}

func (app *Application) execute() {
	asyncErrorChannel := make(chan error)

	app.logger.Info("Starting...", zap.Int("NumCPU", runtime.NumCPU()))

	err := pprofserver.SetupFromViper(asyncErrorChannel, app.v, app.logger)
	if err != nil {
		log.Fatalf("Failed to start net/http/pprof: %v", err)
	}

	app.healthCheck, err = newHealthCheck(app.v, app.logger)
	if err != nil {
		log.Fatalf("Failed to start healthcheck server: %v", err)
	}

	var closeFns []func()
	app.processor, closeFns = startProcessor(app.v, app.logger)

	app.receivers = createReceivers(app.v, app.logger, app.processor)

	err = initTelemetry(asyncErrorChannel, app.v, app.logger)
	if err != nil {
		app.logger.Error("Failed to initialize telemetry", zap.Error(err))
		os.Exit(1)
	}

	signalsChannel := make(chan os.Signal, 1)
	signal.Notify(signalsChannel, os.Interrupt, syscall.SIGTERM)

	// mark service as ready to receive traffic.
	app.healthCheck.Ready()

	select {
	case err = <-asyncErrorChannel:
		app.logger.Error("Asynchronous error received, terminating process", zap.Error(err))
	case s := <-signalsChannel:
		app.logger.Info("Received signal from OS", zap.String("signal", s.String()))
	}

	app.healthCheck.Set(healthcheck.Unavailable)
	app.logger.Info("Starting shutdown...")

	// TODO: orderly shutdown: first receivers, then flushing pipelines giving
	// senders a chance to send all their data. This may take time, the allowed
	// time should be part of configuration.
	for _, receiver := range app.receivers {
		receiver.StopTraceReception(context.Background())
	}

	for _, closeFn := range closeFns {
		closeFn()
	}

	app.logger.Info("Shutdown complete.")
}

// Start the application according to the command and configuration given
// by the user.
func (app *Application) Start() error {
	rootCmd := &cobra.Command{
		Use:  "occollector",
		Long: "OpenCensus Collector",
		Run: func(cmd *cobra.Command, args []string) {
			app.init()
			app.execute()
		},
	}
	viperutils.AddFlags(app.v, rootCmd,
		telemetryFlags,
		builder.Flags,
		healthCheckFlags,
		loggerFlags,
		pprofserver.AddFlags,
	)

	return rootCmd.Execute()
}
