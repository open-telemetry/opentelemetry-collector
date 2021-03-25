// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
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
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configparser"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/internal/collector/telemetry"
	"go.opentelemetry.io/collector/service/internal/builder"
)

const (
	servicezPath   = "servicez"
	pipelinezPath  = "pipelinez"
	extensionzPath = "extensionz"
)

// State defines Application's state.
type State int

const (
	Starting State = iota
	Running
	Closing
	Closed
)

// Application represents a collector application
type Application struct {
	info    component.ApplicationStartInfo
	rootCmd *cobra.Command
	logger  *zap.Logger

	service      *service
	stateChannel chan State

	factories component.Factories

	// stopTestChan is used to terminate the application in end to end tests.
	stopTestChan chan struct{}

	// signalsChannel is used to receive termination signals from the OS.
	signalsChannel chan os.Signal

	// asyncErrorChannel is used to signal a fatal error from any component.
	asyncErrorChannel chan error
}

// Parameters holds configuration for creating a new Application.
type Parameters struct {
	// Factories component factories.
	Factories component.Factories
	// ApplicationStartInfo provides application start information.
	ApplicationStartInfo component.ApplicationStartInfo
	// ConfigFactory that creates the configuration.
	// If it is not provided the default factory (FileLoaderConfigFactory) is used.
	// The default factory loads the configuration file and overrides component's configuration
	// properties supplied via --set command line flag.
	ConfigFactory ConfigFactory
	// LoggingOptions provides a way to change behavior of zap logging.
	LoggingOptions []zap.Option
}

// ConfigFactory creates config.
// The ConfigFactory implementation should call AddSetFlagProperties to enable configuration passed via `--set` flag.
// Viper and command instances are passed from the Application.
// The factories also belong to the Application and are equal to the factories passed via Parameters.
type ConfigFactory func(v *viper.Viper, cmd *cobra.Command, factories component.Factories) (*configmodels.Config, error)

// FileLoaderConfigFactory implements ConfigFactory and it creates configuration from file
// and from --set command line flag (if the flag is present).
func FileLoaderConfigFactory(v *viper.Viper, cmd *cobra.Command, factories component.Factories) (*configmodels.Config, error) {
	file := builder.GetConfigFile()
	if file == "" {
		return nil, errors.New("config file not specified")
	}
	// first load the config file
	v.SetConfigFile(file)
	err := v.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading config file %q: %v", file, err)
	}

	// next overlay the config file with --set flags
	if err := AddSetFlagProperties(v, cmd); err != nil {
		return nil, fmt.Errorf("failed to process set flag: %v", err)
	}
	return configparser.Load(v, factories)
}

// New creates and returns a new instance of Application.
func New(params Parameters) (*Application, error) {
	app := &Application{
		info:         params.ApplicationStartInfo,
		factories:    params.Factories,
		stateChannel: make(chan State, Closed+1),
	}

	factory := params.ConfigFactory
	if factory == nil {
		// use default factory that loads the configuration file
		factory = FileLoaderConfigFactory
	}

	rootCmd := &cobra.Command{
		Use:  params.ApplicationStartInfo.ExeName,
		Long: params.ApplicationStartInfo.LongName,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if app.logger, err = newLogger(params.LoggingOptions); err != nil {
				return fmt.Errorf("failed to get logger: %w", err)
			}

			if err := app.execute(context.Background(), factory); err != nil {
				return err
			}

			return nil
		},
	}

	// TODO: coalesce this code and expose this information to other components.
	flagSet := new(flag.FlagSet)
	addFlagsFns := []func(*flag.FlagSet){
		configtelemetry.Flags,
		telemetry.Flags,
		builder.Flags,
		loggerFlags,
	}
	for _, addFlags := range addFlagsFns {
		addFlags(flagSet)
	}
	rootCmd.Flags().AddGoFlagSet(flagSet)
	addSetFlag(rootCmd.Flags())

	app.rootCmd = rootCmd

	return app, nil
}

// GetStateChannel returns state channel of the application.
func (app *Application) GetStateChannel() chan State {
	return app.stateChannel
}

// Command returns Application's root command.
func (app *Application) Command() *cobra.Command {
	return app.rootCmd
}

// GetLogger returns logger used by the Application.
// The logger is initialized after application start.
func (app *Application) GetLogger() *zap.Logger {
	return app.logger
}

func (app *Application) Shutdown() {
	// TODO: Implement a proper shutdown with graceful draining of the pipeline.
	// See https://github.com/open-telemetry/opentelemetry-collector/issues/483.
	defer func() {
		if r := recover(); r != nil {
			app.logger.Info("stopTestChan already closed")
		}
	}()
	close(app.stopTestChan)
}

func (app *Application) setupTelemetry(ballastSizeBytes uint64) error {
	app.logger.Info("Setting up own telemetry...")

	err := applicationTelemetry.init(app.asyncErrorChannel, ballastSizeBytes, app.logger)
	if err != nil {
		return fmt.Errorf("failed to initialize telemetry: %w", err)
	}

	return nil
}

// runAndWaitForShutdownEvent waits for one of the shutdown events that can happen.
func (app *Application) runAndWaitForShutdownEvent() {
	app.logger.Info("Everything is ready. Begin running and processing data.")

	// plug SIGTERM signal into a channel.
	app.signalsChannel = make(chan os.Signal, 1)
	signal.Notify(app.signalsChannel, os.Interrupt, syscall.SIGTERM)

	// set the channel to stop testing.
	app.stopTestChan = make(chan struct{})
	app.stateChannel <- Running
	select {
	case err := <-app.asyncErrorChannel:
		app.logger.Error("Asynchronous error received, terminating process", zap.Error(err))
	case s := <-app.signalsChannel:
		app.logger.Info("Received signal from OS", zap.String("signal", s.String()))
	case <-app.stopTestChan:
		app.logger.Info("Received stop test request")
	}
	app.stateChannel <- Closing
}

func (app *Application) setupConfigurationComponents(ctx context.Context, factory ConfigFactory) error {
	if err := configcheck.ValidateConfigFromFactories(app.factories); err != nil {
		return err
	}

	app.logger.Info("Loading configuration...")

	cfg, err := factory(configparser.NewViper(), app.rootCmd, app.factories)
	if err != nil {
		return fmt.Errorf("cannot load configuration: %w", err)
	}

	app.logger.Info("Applying configuration...")

	app.service, err = newService(&settings{
		Factories:         app.factories,
		StartInfo:         app.info,
		Config:            cfg,
		Logger:            app.logger,
		AsyncErrorChannel: app.asyncErrorChannel,
	})
	if err != nil {
		return err
	}

	return app.service.Start(ctx)
}

func (app *Application) execute(ctx context.Context, factory ConfigFactory) error {
	app.logger.Info("Starting "+app.info.LongName+"...",
		zap.String("Version", app.info.Version),
		zap.String("GitHash", app.info.GitHash),
		zap.Int("NumCPU", runtime.NumCPU()),
	)
	app.stateChannel <- Starting

	// Set memory ballast
	ballast, ballastSizeBytes := app.createMemoryBallast()

	app.asyncErrorChannel = make(chan error)

	// Setup everything.
	err := app.setupTelemetry(ballastSizeBytes)
	if err != nil {
		return err
	}

	err = app.setupConfigurationComponents(ctx, factory)
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

	if err := app.service.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown service: %w", err))
	}

	if err := applicationTelemetry.shutdown(); err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown application telemetry: %w", err))
	}

	app.logger.Info("Shutdown complete.")
	app.stateChannel <- Closed
	close(app.stateChannel)

	return consumererror.Combine(errs)
}

// Run starts the collector according to the command and configuration
// given by the user, and waits for it to complete.
func (app *Application) Run() error {
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
