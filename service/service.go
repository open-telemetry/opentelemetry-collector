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

// Package service handles the command-line, configuration, and runs the
// OpenTelemetry Service.
package service

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

	"github.com/open-telemetry/opentelemetry-service/config"
	"github.com/open-telemetry/opentelemetry-service/exporter"
	"github.com/open-telemetry/opentelemetry-service/internal/config/viperutils"
	"github.com/open-telemetry/opentelemetry-service/internal/pprofserver"
	"github.com/open-telemetry/opentelemetry-service/internal/zpagesserver"
	"github.com/open-telemetry/opentelemetry-service/processor"
	"github.com/open-telemetry/opentelemetry-service/receiver"
	"github.com/open-telemetry/opentelemetry-service/service/builder"
)

// Application represents a collector application
type Application struct {
	v              *viper.Viper
	logger         *zap.Logger
	healthCheck    *healthcheck.HealthCheck
	exporters      builder.Exporters
	builtReceivers builder.Receivers

	// factories
	receiverFactories  map[string]receiver.Factory
	exporterFactories  map[string]exporter.Factory
	processorFactories map[string]processor.Factory

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

var _ receiver.Host = (*Application)(nil)

// Context returns a context provided by the host to be used on the receiver
// operations.
func (app *Application) Context() context.Context {
	// For now simply the background context.
	return context.Background()
}

// ReportFatalError is used to report to the host that the receiver encountered
// a fatal error (i.e.: an error that the instance can't recover from) after
// its start function has already returned.
func (app *Application) ReportFatalError(err error) {
	app.asyncErrorChannel <- err
}

// OkToIngest returns true when the receiver can inject the received data
// into the pipeline and false when it should drop the data and report
// error to the client.
func (app *Application) OkToIngest() bool {
	return true
}

// New creates and returns a new instance of Application
func New(
	receiverFactories map[string]receiver.Factory,
	processorFactories map[string]processor.Factory,
	exporterFactories map[string]exporter.Factory,
) *Application {
	return &Application{
		v:                  viper.New(),
		readyChan:          make(chan struct{}),
		receiverFactories:  receiverFactories,
		processorFactories: processorFactories,
		exporterFactories:  exporterFactories,
	}
}

func (app *Application) init() {
	file := builder.GetConfigFile(app.v)
	if file == "" {
		log.Fatalf("Config file not specified")
	}
	app.v.SetConfigFile(file)
	err := app.v.ReadInConfig()
	if err != nil {
		log.Fatalf("Error loading config file %q: %v", file, err)
	}
	app.logger, err = newLogger(app.v)
	if err != nil {
		log.Fatalf("Failed to get logger: %v", err)
	}
}

func (app *Application) setupPProf() {
	app.logger.Info("Setting up profiler...")
	err := pprofserver.SetupFromViper(app.asyncErrorChannel, app.v, app.logger)
	if err != nil {
		log.Fatalf("Failed to start net/http/pprof: %v", err)
	}
}

func (app *Application) setupHealthCheck() {
	app.logger.Info("Setting up health checks...")
	var err error
	app.healthCheck, err = newHealthCheck(app.v, app.logger)
	if err != nil {
		log.Fatalf("Failed to start healthcheck server: %v", err)
	}
}

// TODO(ccaraman): Move ZPage configuration to be apart of global config/config.go
func (app *Application) setupZPages() {
	app.logger.Info("Setting up zPages...")
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

func (app *Application) setupTelemetry(ballastSizeBytes uint64) {
	app.logger.Info("Setting up own telemetry...")
	err := AppTelemetry.init(app.asyncErrorChannel, ballastSizeBytes, app.v, app.logger)
	if err != nil {
		app.logger.Error("Failed to initialize telemetry", zap.Error(err))
		os.Exit(1)
	}
}

// runAndWaitForShutdownEvent waits for one of the shutdown events that can happen.
func (app *Application) runAndWaitForShutdownEvent() {
	app.logger.Info("Everything is ready. Begin running and processing data.")

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

func (app *Application) shutdownClosableComponents() {
	for _, closeFn := range app.closeFns {
		closeFn()
	}
}

func (app *Application) setupPipelines() {
	app.logger.Info("Loading configuration...")

	// Load configuration.
	cfg, err := config.Load(app.v, app.receiverFactories, app.processorFactories, app.exporterFactories, app.logger)
	if err != nil {
		log.Fatalf("Cannot load configuration: %v", err)
	}

	app.logger.Info("Applying configuration...")

	// Pipeline is built backwards, starting from exporters, so that we create objects
	// which are referenced before objects which reference them.

	// First create exporters.
	app.exporters, err = builder.NewExportersBuilder(app.logger, cfg, app.exporterFactories).Build()
	if err != nil {
		log.Fatalf("Cannot load configuration: %v", err)
	}

	// Create pipelines and their processors and plug exporters to the
	// end of the pipelines.
	pipelines, err := builder.NewPipelinesBuilder(app.logger, cfg, app.exporters, app.processorFactories).Build()
	if err != nil {
		log.Fatalf("Cannot load configuration: %v", err)
	}

	// Create receivers and plug them into the start of the pipelines.
	app.builtReceivers, err = builder.NewReceiversBuilder(app.logger, cfg, pipelines, app.receiverFactories).Build()
	if err != nil {
		log.Fatalf("Cannot load configuration: %v", err)
	}

	app.logger.Info("Starting receivers...")
	err = app.builtReceivers.StartAll(app.logger, app)
	if err != nil {
		log.Fatalf("Cannot start receivers: %v", err)
	}
}

func (app *Application) shutdownPipelines() {
	// Shutdown order is the reverse of building: first receivers, then flushing pipelines
	// giving senders a chance to send all their data. This may take time, the allowed
	// time should be part of configuration.

	app.logger.Info("Stopping receivers...")
	app.builtReceivers.StopAll()

	// TODO: shutdown processors

	app.exporters.StopAll()
}

func (app *Application) executeUnified() {
	app.logger.Info("Starting...", zap.Int("NumCPU", runtime.NumCPU()))

	// Set memory ballast
	ballast, ballastSizeBytes := app.createMemoryBallast()

	app.asyncErrorChannel = make(chan error)

	// Setup everything.
	app.setupPProf()
	app.setupHealthCheck()
	app.setupZPages()
	app.setupTelemetry(ballastSizeBytes)
	app.setupPipelines()

	// Everything is ready, now run until an event requiring shutdown happens.
	app.runAndWaitForShutdownEvent()

	// Begin shutdown sequence.
	runtime.KeepAlive(ballast)
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
		Use:  "otelsvc",
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

func (app *Application) createMemoryBallast() ([]byte, uint64) {
	ballastSizeMiB := builder.MemBallastSize(app.v)
	if ballastSizeMiB > 0 {
		ballastSizeBytes := uint64(ballastSizeMiB) * 1024 * 1024
		ballast := make([]byte, ballastSizeBytes)
		app.logger.Info("Using memory ballast", zap.Int("MiBs", ballastSizeMiB))
		return ballast, ballastSizeBytes
	}
	return nil, 0
}
