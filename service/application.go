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
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configloader"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/config/experimental/configsource"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/internal/collector/telemetry"
	"go.opentelemetry.io/collector/service/internal/builder"
	"go.opentelemetry.io/collector/service/parserprovider"
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
	info    component.BuildInfo
	rootCmd *cobra.Command
	logger  *zap.Logger

	service      *service
	stateChannel chan State

	factories component.Factories

	parserProvider parserprovider.ParserProvider

	// stopTestChan is used to terminate the application in end to end tests.
	stopTestChan chan struct{}

	// signalsChannel is used to receive termination signals from the OS.
	signalsChannel chan os.Signal

	// asyncErrorChannel is used to signal a fatal error from any component.
	asyncErrorChannel chan error
}

// New creates and returns a new instance of Application.
func New(set AppSettings) (*Application, error) {
	if err := configcheck.ValidateConfigFromFactories(set.Factories); err != nil {
		return nil, err
	}

	app := &Application{
		info:         set.BuildInfo,
		factories:    set.Factories,
		stateChannel: make(chan State, Closed+1),
	}

	rootCmd := &cobra.Command{
		Use:     set.BuildInfo.Command,
		Version: set.BuildInfo.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if app.logger, err = newLogger(set.LoggingOptions); err != nil {
				return fmt.Errorf("failed to get logger: %w", err)
			}

			return app.execute(context.Background())
		},
	}

	// TODO: coalesce this code and expose this information to other components.
	flagSet := new(flag.FlagSet)
	addFlagsFns := []func(*flag.FlagSet){
		configtelemetry.Flags,
		parserprovider.Flags,
		telemetry.Flags,
		builder.Flags,
		loggerFlags,
	}
	for _, addFlags := range addFlagsFns {
		addFlags(flagSet)
	}
	rootCmd.Flags().AddGoFlagSet(flagSet)
	app.rootCmd = rootCmd

	parserProvider := set.ParserProvider
	if parserProvider == nil {
		// use default provider.
		parserProvider = parserprovider.Default()
	}
	app.parserProvider = parserProvider

	return app, nil
}

// Run starts the collector according to the command and configuration
// given by the user, and waits for it to complete.
func (app *Application) Run() error {
	// From this point on do not show usage in case of error.
	app.rootCmd.SilenceUsage = true

	return app.rootCmd.Execute()
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

// Shutdown shuts down the application.
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

// setupConfigurationComponents loads the config and starts the components. If all the steps succeeds it
// sets the app.service with the service currently running.
func (app *Application) setupConfigurationComponents(ctx context.Context) error {
	app.logger.Info("Loading configuration...")

	cp, err := app.parserProvider.Get()
	if err != nil {
		return fmt.Errorf("cannot load configuration's parser: %w", err)
	}

	cfg, err := configloader.Load(cp, app.factories)
	if err != nil {
		return fmt.Errorf("cannot load configuration: %w", err)
	}

	if err = cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	app.logger.Info("Applying configuration...")

	service, err := newService(&svcSettings{
		BuildInfo:         app.info,
		Factories:         app.factories,
		Config:            cfg,
		Logger:            app.logger,
		AsyncErrorChannel: app.asyncErrorChannel,
	})
	if err != nil {
		return err
	}

	err = service.Start(ctx)
	if err != nil {
		return err
	}

	app.service = service

	// If provider is watchable start a goroutine watching for updates.
	if watchable, ok := app.parserProvider.(parserprovider.Watchable); ok {
		go func() {
			err := watchable.WatchForUpdate()
			switch {
			// TODO: Move configsource.ErrSessionClosed to providerparser package to avoid depending on configsource.
			case errors.Is(err, configsource.ErrSessionClosed):
				// This is the case of shutdown of the whole application, nothing to do.
				app.logger.Info("Config WatchForUpdate closed", zap.Error(err))
				return
			default:
				app.logger.Warn("Config WatchForUpdated exited", zap.Error(err))
				app.reloadService(context.Background())
			}
		}()
	}

	return nil
}

func (app *Application) execute(ctx context.Context) error {
	app.logger.Info("Starting "+app.info.Command+"...",
		zap.String("Version", app.info.Version),
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

	err = app.setupConfigurationComponents(ctx)
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

	if closable, ok := app.parserProvider.(parserprovider.Closeable); ok {
		if err := closable.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to close config: %w", err))
		}
	}

	if app.service != nil {
		if err := app.service.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown service: %w", err))
		}
	}

	if err := applicationTelemetry.shutdown(); err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown application telemetry: %w", err))
	}

	app.logger.Info("Shutdown complete.")
	app.stateChannel <- Closed
	close(app.stateChannel)

	return consumererror.Combine(errs)
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

// reloadService shutdowns the current app.service and setups a new one according
// to the latest configuration. It requires that app.parserProvider and app.factories
// are properly populated to finish successfully.
func (app *Application) reloadService(ctx context.Context) error {
	if closeable, ok := app.parserProvider.(parserprovider.Closeable); ok {
		if err := closeable.Close(ctx); err != nil {
			return fmt.Errorf("failed close current config provider: %w", err)
		}
	}

	if app.service != nil {
		retiringService := app.service
		app.service = nil
		if err := retiringService.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown the retiring config: %w", err)
		}
	}

	if err := app.setupConfigurationComponents(ctx); err != nil {
		return fmt.Errorf("failed to setup configuration components: %w", err)
	}

	return nil
}
