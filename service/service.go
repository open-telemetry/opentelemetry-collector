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
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config"
	"github.com/open-telemetry/opentelemetry-collector/config/configcheck"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
	"github.com/open-telemetry/opentelemetry-collector/service/builder"
)

// Application represents a collector application
type Application struct {
	info           ApplicationStartInfo
	rootCmd        *cobra.Command
	v              *viper.Viper
	logger         *zap.Logger
	exporters      builder.Exporters
	builtReceivers builder.Receivers
	builtPipelines builder.BuiltPipelines

	factories config.Factories
	config    *configmodels.Config

	extensions []component.ServiceExtension

	// stopTestChan is used to terminate the application in end to end tests.
	stopTestChan chan struct{}
	// readyChan is used in tests to indicate that the application is ready.
	readyChan chan struct{}

	// asyncErrorChannel is used to signal a fatal error from any component.
	asyncErrorChannel chan error
}

// Command returns Application's root command.
func (app *Application) Command() *cobra.Command {
	return app.rootCmd
}

// ApplicationStartInfo is the information that is logged at the application start.
// This information can be overridden in custom builds.
type ApplicationStartInfo struct {
	// Executable file name, e.g. "otelcol".
	ExeName string

	// Long name, used e.g. in the logs.
	LongName string

	// Version string.
	Version string

	// Git hash of the source code.
	GitHash string
}

// Parameters holds configuration for creating a new Application.
type Parameters struct {
	// Factories component factories.
	Factories config.Factories
	// ApplicationStartInfo provides application start information.
	ApplicationStartInfo ApplicationStartInfo
	// ConfigFactory that creates the configuration.
	// If it is not provided the default factory will be used.
	// The default factory loads the configuration specified as a command line flag.
	ConfigFactory ConfigFactory
}

// ConfigFactory creates config.
type ConfigFactory func(v *viper.Viper, factories config.Factories) (*configmodels.Config, error)

func fileLoaderConfigFactory(v *viper.Viper, factories config.Factories) (*configmodels.Config, error) {
	file := builder.GetConfigFile()
	if file == "" {
		return nil, errors.New("config file not specified")
	}
	v.SetConfigFile(file)
	err := v.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading config file %q: %v", file, err)
	}
	return config.Load(v, factories)
}

// New creates and returns a new instance of Application.
func New(params Parameters) (*Application, error) {
	app := &Application{
		info:      params.ApplicationStartInfo,
		v:         config.NewViper(),
		readyChan: make(chan struct{}),
		factories: params.Factories,
	}

	factory := params.ConfigFactory
	if factory == nil {
		// use default factory that loads the configuration file
		factory = fileLoaderConfigFactory
	}

	rootCmd := &cobra.Command{
		Use:  params.ApplicationStartInfo.ExeName,
		Long: params.ApplicationStartInfo.LongName,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := app.init()
			if err != nil {
				return err
			}

			err = app.execute(factory)
			if err != nil {
				return err
			}

			return nil
		},
	}

	// TODO: coalesce this code and expose this information to other components.
	flagSet := new(flag.FlagSet)
	addFlagsFns := []func(*flag.FlagSet){
		telemetryFlags,
		builder.Flags,
		loggerFlags,
	}
	for _, addFlags := range addFlagsFns {
		addFlags(flagSet)
	}
	rootCmd.Flags().AddGoFlagSet(flagSet)

	app.rootCmd = rootCmd

	return app, nil
}

// ReportFatalError is used to report to the host that the receiver encountered
// a fatal error (i.e.: an error that the instance can't recover from) after
// its start function has already returned.
func (app *Application) ReportFatalError(err error) {
	app.asyncErrorChannel <- err
}

func (app *Application) GetFactory(kind component.Kind, componentType string) component.Factory {
	switch kind {
	case component.KindReceiver:
		return app.factories.Receivers[componentType]
	case component.KindProcessor:
		return app.factories.Processors[componentType]
	case component.KindExporter:
		return app.factories.Exporters[componentType]
	case component.KindExtension:
		return app.factories.Extensions[componentType]
	}
	return nil
}

func (app *Application) init() error {
	l, err := newLogger()
	if err != nil {
		return errors.Wrap(err, "failed to get logger")
	}
	app.logger = l
	return nil
}

func (app *Application) setupTelemetry(ballastSizeBytes uint64) error {
	app.logger.Info("Setting up own telemetry...")

	err := AppTelemetry.init(app.asyncErrorChannel, ballastSizeBytes, app.logger)
	if err != nil {
		return errors.Wrap(err, "failed to initialize telemetry")
	}

	return nil
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

func (app *Application) setupConfigurationComponents(factory ConfigFactory) error {
	if err := configcheck.ValidateConfigFromFactories(app.factories); err != nil {
		return err
	}

	app.logger.Info("Loading configuration...")
	cfg, err := factory(app.v, app.factories)
	if err != nil {
		return errors.Wrap(err, "cannot load configuration")
	}
	err = config.ValidateConfig(cfg, app.logger)
	if err != nil {
		return errors.Wrap(err, "cannot load configuration")
	}

	app.config = cfg
	app.logger.Info("Applying configuration...")

	err = app.setupExtensions()
	if err != nil {
		return errors.Wrap(err, "cannot setup extensions")
	}

	err = app.setupPipelines()
	if err != nil {
		return errors.Wrap(err, "cannot setup pipelines")
	}

	return nil
}

func (app *Application) setupExtensions() error {
	for _, extName := range app.config.Service.Extensions {
		extCfg, exists := app.config.Extensions[extName]
		if !exists {
			return errors.Errorf("extension %q is not configured", extName)
		}

		factory, exists := app.factories.Extensions[extCfg.Type()]
		if !exists {
			return errors.Errorf("extension factory for type %q is not configured", extCfg.Type())
		}

		ext, err := factory.CreateExtension(app.logger, extCfg)
		if err != nil {
			return errors.Wrapf(err, "failed to create extension %q", extName)
		}

		// Check if the factory really created the extension.
		if ext == nil {
			return errors.Errorf("factory for %q produced a nil extension", extName)
		}

		if err := ext.Start(app); err != nil {
			return errors.Wrapf(err, "error starting extension %q", extName)
		}

		app.extensions = append(app.extensions, ext)
	}

	return nil
}

func (app *Application) setupPipelines() error {
	// Pipeline is built backwards, starting from exporters, so that we create objects
	// which are referenced before objects which reference them.

	// First create exporters.
	var err error
	app.exporters, err = builder.NewExportersBuilder(app.logger, app.config, app.factories.Exporters).Build()
	if err != nil {
		return errors.Wrap(err, "cannot build exporters")
	}
	app.logger.Info("Starting exporters...")
	err = app.exporters.StartAll(app.logger, app)
	if err != nil {
		return errors.Wrap(err, "cannot start exporters")
	}

	// Create pipelines and their processors and plug exporters to the
	// end of the pipelines.
	app.builtPipelines, err = builder.NewPipelinesBuilder(app.logger, app.config, app.exporters, app.factories.Processors).Build()
	if err != nil {
		return errors.Wrap(err, "cannot build pipelines")
	}

	app.logger.Info("Starting processors...")
	err = app.builtPipelines.StartProcessors(app.logger, app)
	if err != nil {
		return errors.Wrap(err, "cannot start processors")
	}

	// Create receivers and plug them into the start of the pipelines.
	app.builtReceivers, err = builder.NewReceiversBuilder(app.logger, app.config, app.builtPipelines, app.factories.Receivers).Build()
	if err != nil {
		return errors.Wrap(err, "cannot build receivers")
	}

	app.logger.Info("Starting receivers...")
	err = app.builtReceivers.StartAll(app.logger, app)
	if err != nil {
		return errors.Wrap(err, "cannot start receivers")
	}

	return nil
}

func (app *Application) notifyPipelineReady() error {
	for i, ext := range app.extensions {
		if pw, ok := ext.(component.PipelineWatcher); ok {
			if err := pw.Ready(); err != nil {
				return errors.Wrapf(
					err,
					"error notifying extension %q that the pipeline was started",
					app.config.Service.Extensions[i],
				)
			}
		}
	}

	return nil
}

func (app *Application) notifyPipelineNotReady() error {
	// Notify on reverse order.
	var errs []error
	for i := len(app.extensions) - 1; i >= 0; i-- {
		ext := app.extensions[i]
		if pw, ok := ext.(component.PipelineWatcher); ok {
			if err := pw.NotReady(); err != nil {
				errs = append(errs, errors.Wrapf(err,
					"error notifying extension %q that the pipeline was shutdown",
					app.config.Service.Extensions[i]))
			}
		}
	}

	if len(errs) != 0 {
		return oterr.CombineErrors(errs)
	}

	return nil
}

func (app *Application) shutdownPipelines() error {
	// Shutdown order is the reverse of building: first receivers, then flushing pipelines
	// giving senders a chance to send all their data. This may take time, the allowed
	// time should be part of configuration.

	var errs []error

	app.logger.Info("Stopping receivers...")
	err := app.builtReceivers.StopAll()
	if err != nil {
		errs = append(errs, errors.Wrap(err, "failed to stop receivers"))
	}

	app.logger.Info("Stopping processors...")
	err = app.builtPipelines.ShutdownProcessors(app.logger)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "failed to shutdown processors"))
	}

	app.logger.Info("Shutting down exporters...")
	err = app.exporters.ShutdownAll()
	if err != nil {
		errs = append(errs, errors.Wrap(err, "failed to shutdown exporters"))
	}

	if len(errs) != 0 {
		return oterr.CombineErrors(errs)
	}

	return nil
}

func (app *Application) shutdownExtensions() error {
	// Shutdown on reverse order.
	var errs []error
	for i := len(app.extensions) - 1; i >= 0; i-- {
		ext := app.extensions[i]
		if err := ext.Shutdown(); err != nil {
			errs = append(errs, errors.Wrapf(err,
				"error shutting down extension %q",
				app.config.Service.Extensions[i]))
		}
	}

	if len(errs) != 0 {
		return oterr.CombineErrors(errs)
	}

	return nil
}

func (app *Application) execute(factory ConfigFactory) error {
	app.logger.Info("Starting "+app.info.LongName+"...",
		zap.String("Version", app.info.Version),
		zap.String("GitHash", app.info.GitHash),
		zap.Int("NumCPU", runtime.NumCPU()),
	)

	// Set memory ballast
	ballast, ballastSizeBytes := app.createMemoryBallast()

	app.asyncErrorChannel = make(chan error)

	// Setup everything.
	err := app.setupTelemetry(ballastSizeBytes)
	if err != nil {
		return err
	}

	err = app.setupConfigurationComponents(factory)
	if err != nil {
		return err
	}

	err = app.notifyPipelineReady()
	if err != nil {
		return err
	}

	// Everything is ready, now run until an event requiring shutdown happens.
	app.runAndWaitForShutdownEvent()

	// Accumulate errors and proceed with shutting down remaining components.
	var errs []error

	// Begin shutdown sequence.
	runtime.KeepAlive(ballast)
	app.logger.Info("Starting shutdown...")

	err = app.notifyPipelineNotReady()
	if err != nil {
		errs = append(errs, errors.Wrap(err, "failed to notify that pipeline is not ready"))
	}

	err = app.shutdownPipelines()
	if err != nil {
		errs = append(errs, errors.Wrap(err, "failed to shutdown pipelines"))
	}

	err = app.shutdownExtensions()
	if err != nil {
		errs = append(errs, errors.Wrap(err, "failed to shutdown extensions"))
	}

	AppTelemetry.shutdown()

	app.logger.Info("Shutdown complete.")

	if len(errs) != 0 {
		return oterr.CombineErrors(errs)
	}
	return nil
}

// Start starts the collector according to the command and configuration
// given by the user.
func (app *Application) Start() error {
	// From this point on do not show usage in case of error.
	app.rootCmd.SilenceUsage = true

	return app.rootCmd.Execute()
}

func (app *Application) createMemoryBallast() ([]byte, uint64) {
	ballastSizeMiB := builder.MemBallastSize()
	if ballastSizeMiB > 0 {
		ballastSizeBytes := uint64(ballastSizeMiB) * 1024 * 1024
		ballast := make([]byte, ballastSizeBytes)
		app.logger.Info("Using memory ballast", zap.Int("MiBs", ballastSizeMiB))
		return ballast, ballastSizeBytes
	}
	return nil, 0
}
