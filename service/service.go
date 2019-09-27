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
// OpenTelemetry Collector.
package service

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/extension"
	"github.com/open-telemetry/opentelemetry-collector/internal/config/viperutils"
	"github.com/open-telemetry/opentelemetry-collector/receiver"
	"github.com/open-telemetry/opentelemetry-collector/service/builder"
)

// Application represents a collector application
type Application struct {
	v              *viper.Viper
	logger         *zap.Logger
	exporters      builder.Exporters
	builtReceivers builder.Receivers

	factories config.Factories
	config    *configmodels.Config

	extensions []extension.ServiceExtension

	// stopTestChan is used to terminate the application in end to end tests.
	stopTestChan chan struct{}
	// readyChan is used in tests to indicate that the application is ready.
	readyChan chan struct{}

	// asyncErrorChannel is used to signal a fatal error from any component.
	asyncErrorChannel chan error
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

// New creates and returns a new instance of Application
func New(
	factories config.Factories,
) *Application {
	return &Application{
		v:         viper.New(),
		readyChan: make(chan struct{}),
		factories: factories,
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

func (app *Application) setupConfigurationComponents() {
	// Load configuration.
	app.logger.Info("Loading configuration...")
	cfg, err := config.Load(app.v, app.factories, app.logger)
	if err != nil {
		log.Fatalf("Cannot load configuration: %v", err)
	}

	app.config = cfg

	app.logger.Info("Applying configuration...")

	app.setupExtensions()
	app.setupPipelines()
}

func (app *Application) setupExtensions() {
	for _, extName := range app.config.Service.Extensions {
		extCfg, exists := app.config.Extensions[extName]
		if !exists {
			log.Fatalf("Cannot load configuration: extension %q is not configured", extName)
		}

		factory, exists := app.factories.Extensions[extCfg.Type()]
		if !exists {
			log.Fatalf("Cannot load configuration: extension factory for type %q is not configured", extCfg.Type())
		}

		ext, err := factory.CreateExtension(app.logger, extCfg)
		if err != nil {
			log.Fatalf("Cannot load configuration: failed to create extension %q: %v", extName, err)
		}

		if err := ext.Start(app); err != nil {
			log.Fatalf("Cannot start extension %q: %v", extName, err)
		}
		app.extensions = append(app.extensions, ext)
	}
}

func (app *Application) setupPipelines() {
	// Pipeline is built backwards, starting from exporters, so that we create objects
	// which are referenced before objects which reference them.

	// First create exporters.
	var err error
	app.exporters, err = builder.NewExportersBuilder(app.logger, app.config, app.factories.Exporters).Build()
	if err != nil {
		log.Fatalf("Cannot load configuration: %v", err)
	}

	// Create pipelines and their processors and plug exporters to the
	// end of the pipelines.
	pipelines, err := builder.NewPipelinesBuilder(app.logger, app.config, app.exporters, app.factories.Processors).Build()
	if err != nil {
		log.Fatalf("Cannot load configuration: %v", err)
	}

	// Create receivers and plug them into the start of the pipelines.
	app.builtReceivers, err = builder.NewReceiversBuilder(app.logger, app.config, pipelines, app.factories.Receivers).Build()
	if err != nil {
		log.Fatalf("Cannot load configuration: %v", err)
	}

	app.logger.Info("Starting receivers...")
	err = app.builtReceivers.StartAll(app.logger, app)
	if err != nil {
		log.Fatalf("Cannot start receivers: %v", err)
	}
}

func (app *Application) notifyPipelineReady() {
	for i, ext := range app.extensions {
		if pw, ok := ext.(extension.PipelineWatcher); ok {
			if err := pw.Ready(); err != nil {
				log.Fatalf(
					"Error notifying extension %q that the pipeline was started: %v",
					app.config.Service.Extensions[i],
					err,
				)
			}
		}
	}
}

func (app *Application) notifyPipelineNotReady() {
	// Notify on reverse order.
	for i := len(app.extensions) - 1; i >= 0; i-- {
		ext := app.extensions[i]
		if pw, ok := ext.(extension.PipelineWatcher); ok {
			if err := pw.NotReady(); err != nil {
				app.logger.Warn(
					"Error notifying extension that the pipeline was shutdown",
					zap.Error(err),
					zap.String("extension", app.config.Service.Extensions[i]),
				)
			}
		}
	}
}

func (app *Application) shutdownPipelines() {
	// Shutdown order is the reverse of building: first receivers, then flushing pipelines
	// giving senders a chance to send all their data. This may take time, the allowed
	// time should be part of configuration.

	app.logger.Info("Stopping receivers...")
	app.builtReceivers.StopAll()

	// TODO: shutdown processors

	app.logger.Info("Shutting down exporters...")
	app.exporters.ShutdownAll()
}

func (app *Application) shutdownExtensions() {
	// Shutdown on reverse order.
	for i := len(app.extensions) - 1; i >= 0; i-- {
		ext := app.extensions[i]
		if err := ext.Shutdown(); err != nil {
			app.logger.Warn(
				"Error shutting down extension",
				zap.Error(err),
				zap.String("extension", app.config.Service.Extensions[i]),
			)
		}
	}
}

func (app *Application) executeUnified() {
	app.logger.Info("Starting...", zap.Int("NumCPU", runtime.NumCPU()))

	// Set memory ballast
	ballast, ballastSizeBytes := app.createMemoryBallast()

	app.asyncErrorChannel = make(chan error)

	// Setup everything.
	app.setupTelemetry(ballastSizeBytes)
	app.setupConfigurationComponents()
	app.notifyPipelineReady()

	// Everything is ready, now run until an event requiring shutdown happens.
	app.runAndWaitForShutdownEvent()

	// Begin shutdown sequence.
	runtime.KeepAlive(ballast)
	app.logger.Info("Starting shutdown...")

	app.notifyPipelineNotReady()
	app.shutdownPipelines()
	app.shutdownExtensions()

	AppTelemetry.shutdown()

	app.logger.Info("Shutdown complete.")
}

// StartUnified starts the unified service according to the command and configuration
// given by the user.
func (app *Application) StartUnified() error {
	rootCmd := &cobra.Command{
		Use:  "otelcol",
		Long: "OpenTelemetry Collector",
		Run: func(cmd *cobra.Command, args []string) {
			app.init()
			app.executeUnified()
		},
	}
	viperutils.AddFlags(app.v, rootCmd,
		telemetryFlags,
		builder.Flags,
		loggerFlags,
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
