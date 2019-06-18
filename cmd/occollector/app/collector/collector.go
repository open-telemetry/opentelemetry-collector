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

	"github.com/open-telemetry/opentelemetry-service/cmd/occollector/app/builder"
	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/internal/config/viperutils"
	"github.com/open-telemetry/opentelemetry-service/internal/configv2"
	"github.com/open-telemetry/opentelemetry-service/internal/pprofserver"
	"github.com/open-telemetry/opentelemetry-service/internal/zpagesserver"
	"github.com/open-telemetry/opentelemetry-service/receiver"
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
	exporters   builder.Exporters

	// stopTestChan is used to terminate the application in end to end tests.
	stopTestChan chan struct{}
	// readyChan is used in tests to indicate that the application is ready.
	readyChan chan struct{}

	// asyncErrorChannel is used to signal a fatal error from any component.
	asyncErrorChannel chan error

	// closeFns are functions that must be called on application shutdown.
	// Various components can add their own functions that they need to be
	// called for cleanup during shutdown.
	closeFns []func()
}

func newApp() *Application {
	return &Application{
		v:         viper.New(),
		readyChan: make(chan struct{}),
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

func (app *Application) setupPProf() {
	err := pprofserver.SetupFromViper(app.asyncErrorChannel, app.v, app.logger)
	if err != nil {
		log.Fatalf("Failed to start net/http/pprof: %v", err)
	}
}

func (app *Application) setupHealthCheck() {
	var err error
	app.healthCheck, err = newHealthCheck(app.v, app.logger)
	if err != nil {
		log.Fatalf("Failed to start healthcheck server: %v", err)
	}
}

func (app *Application) setupZPages() {
	zpagesPort := app.v.GetInt(zpagesserver.ZPagesHTTPPort)
	if zpagesPort > 0 {
		closeZPages, err := zpagesserver.Run(app.asyncErrorChannel, zpagesPort)
		if err != nil {
			app.logger.Error("Failed to run zPages", zap.Error(err))
			os.Exit(1)
		}
		app.logger.Info("Running zPages", zap.Int("port", zpagesPort))
		closeFn := func() {
			closeZPages()
		}
		app.closeFns = append(app.closeFns, closeFn)
	}
}

func (app *Application) setupTelemetry() {
	err := AppTelemetry.init(app.asyncErrorChannel, app.v, app.logger)
	if err != nil {
		app.logger.Error("Failed to initialize telemetry", zap.Error(err))
		os.Exit(1)
	}
}

// runAndWaitForShutdownEvent waits for one of the shutdown events that can happen.
func (app *Application) runAndWaitForShutdownEvent() {
	// Plug SIGTERM signal into a channel.
	signalsChannel := make(chan os.Signal, 1)
	signal.Notify(signalsChannel, os.Interrupt, syscall.SIGTERM)

	// mark service as ready to receive traffic.
	app.healthCheck.Ready()

	// set the channel to stop testing.
	app.stopTestChan = make(chan struct{})
	// notify tests that it is ready.
	close(app.readyChan)

	select {
	case err := <-app.asyncErrorChannel:
		app.logger.Error("Asynchronous error received, terminating process", zap.Error(err))
	case s := <-signalsChannel:
		app.logger.Info("Received signal from OS", zap.String("signal", s.String()))
	case <-app.stopTestChan:
		app.logger.Info("Received stop test request")
	}
}

func (app *Application) shutdownReceivers() {
	for _, receiver := range app.receivers {
		receiver.StopTraceReception(context.Background())
	}
}

func (app *Application) shutdownClosableComponents() {
	for _, closeFn := range app.closeFns {
		closeFn()
	}
}

func (app *Application) execute() {
	app.logger.Info("Starting...", zap.Int("NumCPU", runtime.NumCPU()))

	app.asyncErrorChannel = make(chan error)

	// Setup everything.

	app.setupPProf()
	app.setupHealthCheck()
	app.processor, app.closeFns = startProcessor(app.v, app.logger)
	app.setupZPages()
	app.receivers = createReceivers(app.v, app.logger, app.processor, app.asyncErrorChannel)
	app.setupTelemetry()

	// Everything is ready, now run until an event requiring shutdown happens.

	app.runAndWaitForShutdownEvent()

	// Begin shutdown sequence.

	app.healthCheck.Set(healthcheck.Unavailable)
	app.logger.Info("Starting shutdown...")

	// TODO: orderly shutdown: first receivers, then flushing pipelines giving
	// senders a chance to send all their data. This may take time, the allowed
	// time should be part of configuration.
	app.shutdownReceivers()

	app.shutdownClosableComponents()

	AppTelemetry.shutdown()

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
		zpagesserver.AddFlags,
	)

	return rootCmd.Execute()
}

func (app *Application) setupPipelines() {
	// Load configuration.
	config, err := configv2.Load(app.v)
	if err != nil {
		log.Fatalf("Cannot load configuration: %v", err)
	}

	// Pipeline is built backwards, starting from exporters, so that we create objects
	// which are referenced before objects which reference them.

	// First create exporters.
	app.exporters, err = builder.NewExportersBuilder(app.logger, config).Build()
	if err != nil {
		log.Fatalf("Cannot load configuration: %v", err)
	}

	// Create pipelines and their processors and plug exporters to the
	// end of the pipelines.
	_, err = builder.NewPipelinesBuilder(app.logger, config, app.exporters).Build()
	if err != nil {
		log.Fatalf("Cannot load configuration: %v", err)
	}

	// TODO: create receivers and plug them into the start of the pipelines.
}

func (app *Application) shutdownPipelines() {
	// Shutdown order is the reverse of building: first receivers, then flushing pipelines
	// giving senders a chance to send all their data. This may take time, the allowed
	// time should be part of configuration.

	// TODO: shutdown receivers.

	// TODO: shutdown processors

	app.exporters.StopAll()
}

func (app *Application) executeUnified() {
	app.logger.Info("Starting...", zap.Int("NumCPU", runtime.NumCPU()))

	app.asyncErrorChannel = make(chan error)

	// Setup everything.

	app.setupPProf()
	app.setupHealthCheck()
	app.setupZPages()
	app.setupTelemetry()
	app.setupPipelines()

	// Everything is ready, now run until an event requiring shutdown happens.

	app.runAndWaitForShutdownEvent()

	// Begin shutdown sequence.

	app.healthCheck.Set(healthcheck.Unavailable)
	app.logger.Info("Starting shutdown...")

	app.shutdownPipelines()
	app.shutdownClosableComponents()

	AppTelemetry.shutdown()

	app.logger.Info("Shutdown complete.")
}

// StartUnified starts the unified service according to the command and configuration
// given by the user.
func (app *Application) StartUnified() error {
	rootCmd := &cobra.Command{
		Use:  "unisvc",
		Long: "OpenTelemetry Service",
		Run: func(cmd *cobra.Command, args []string) {
			app.init()
			app.executeUnified()
		},
	}
	viperutils.AddFlags(app.v, rootCmd,
		telemetryFlags,
		builder.Flags,
		healthCheckFlags,
		loggerFlags,
		pprofserver.AddFlags,
		zpagesserver.AddFlags,
	)

	return rootCmd.Execute()
}
