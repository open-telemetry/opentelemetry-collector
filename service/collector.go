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
	"go.opentelemetry.io/contrib/zpages"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/config/configunmarshaler"
	"go.opentelemetry.io/collector/config/experimental/configsource"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/extension/ballastextension"
	"go.opentelemetry.io/collector/internal/collector/telemetry"
	"go.opentelemetry.io/collector/service/internal"
	"go.opentelemetry.io/collector/service/internal/builder"
	"go.opentelemetry.io/collector/service/parserprovider"
)

// State defines Collector's state.
type State int

const (
	Starting State = iota
	Running
	Closing
	Closed
)

// (Internal note) Collector Lifecycle:
// - New constructs a new Collector.
// - Run starts the collector and calls (*Collector).execute.
// - execute calls setupConfigurationComponents to handle configuration.
//   If configuration parser fails, collector's config can be reloaded.
//   Collector can be shutdown if parser gets a shutdown error.
// - execute runs runAndWaitForShutdownEvent and waits for a shutdown event.
//   SIGINT and SIGTERM, errors, and (*Collector).Shutdown can trigger the shutdown events.
// - Upon shutdown, pipelines are notified, then pipelines and extensions are shut down.
// - Users can call (*Collector).Shutdown anytime to shutdown the collector.

// Collector represents a server providing the OpenTelemetry Collector service.
type Collector struct {
	info    component.BuildInfo
	rootCmd *cobra.Command
	logger  *zap.Logger

	tracerProvider      trace.TracerProvider
	zPagesSpanProcessor *zpages.SpanProcessor

	service      *service
	stateChannel chan State

	factories component.Factories

	parserProvider    parserprovider.ParserProvider
	configUnmarshaler configunmarshaler.ConfigUnmarshaler

	// shutdownChan is used to terminate the collector.
	shutdownChan chan struct{}

	// signalsChannel is used to receive termination signals from the OS.
	signalsChannel chan os.Signal

	allowGracefulShutodwn bool

	// asyncErrorChannel is used to signal a fatal error from any component.
	asyncErrorChannel chan error
}

// New creates and returns a new instance of Collector.
func New(set CollectorSettings) (*Collector, error) {
	if err := configcheck.ValidateConfigFromFactories(set.Factories); err != nil {
		return nil, err
	}

	col := &Collector{
		info:              set.BuildInfo,
		factories:         set.Factories,
		stateChannel:      make(chan State, Closed+1),
		parserProvider:    set.ParserProvider,
		configUnmarshaler: set.ConfigUnmarshaler,
		// We use a negative in the settings not to break the existing
		// behavior. Internally, allowGracefulShutodwn is more readable.
		allowGracefulShutodwn: !set.DisableGracefulShutdown,
	}

	if col.parserProvider == nil {
		// use default provider.
		col.parserProvider = parserprovider.Default()
	}

	if col.configUnmarshaler == nil {
		// use default provider.
		col.configUnmarshaler = configunmarshaler.NewDefault()
	}

	rootCmd := &cobra.Command{
		Use:     set.BuildInfo.Command,
		Version: set.BuildInfo.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if col.logger, err = newLogger(set.LoggingOptions); err != nil {
				return fmt.Errorf("failed to get logger: %w", err)
			}

			col.zPagesSpanProcessor = zpages.NewSpanProcessor()
			col.tracerProvider = sdktrace.NewTracerProvider(
				sdktrace.WithSampler(internal.AlwaysRecord()),
				sdktrace.WithSpanProcessor(col.zPagesSpanProcessor))

			// Set the constructed tracer provider as Global, in case any component uses the
			// global TracerProvider.
			otel.SetTracerProvider(col.tracerProvider)

			return col.execute(cmd.Context())
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
	col.rootCmd = rootCmd

	return col, nil
}

// Run starts the collector according to the command and configuration
// given by the user, and waits for it to complete.
// Consecutive calls to Run are not allowed, Run shouldn't be called
// once a collector is shut down.
func (col *Collector) Run() error {
	// From this point on do not show usage in case of error.
	col.rootCmd.SilenceUsage = true

	return col.rootCmd.Execute()
}

// GetStateChannel returns state channel of the collector server.
func (col *Collector) GetStateChannel() chan State {
	return col.stateChannel
}

// Command returns Collector's root command.
func (col *Collector) Command() *cobra.Command {
	return col.rootCmd
}

// GetLogger returns logger used by the Collector.
// The logger is initialized after collector server start.
func (col *Collector) GetLogger() *zap.Logger {
	return col.logger
}

// Shutdown shuts down the collector server.
func (col *Collector) Shutdown() {
	// TODO: Implement a proper shutdown with graceful draining of the pipeline.
	// See https://github.com/open-telemetry/opentelemetry-collector/issues/483.
	defer func() {
		if r := recover(); r != nil {
			col.logger.Info("shutdownChan already closed")
		}
	}()
	close(col.shutdownChan)
}

func (col *Collector) setupTelemetry(ballastSizeBytes uint64) error {
	col.logger.Info("Setting up own telemetry...")

	err := collectorTelemetry.init(col.asyncErrorChannel, ballastSizeBytes, col.logger)
	if err != nil {
		return fmt.Errorf("failed to initialize telemetry: %w", err)
	}

	return nil
}

// runAndWaitForShutdownEvent waits for one of the shutdown events that can happen.
func (col *Collector) runAndWaitForShutdownEvent() {
	col.logger.Info("Everything is ready. Begin running and processing data.")

	col.signalsChannel = make(chan os.Signal, 1)
	// Only notify with SIGTERM and SIGINT if graceful shutdown is enabled.
	if col.allowGracefulShutodwn {
		signal.Notify(col.signalsChannel, os.Interrupt, syscall.SIGTERM)
	}

	col.shutdownChan = make(chan struct{})
	col.stateChannel <- Running
	select {
	case err := <-col.asyncErrorChannel:
		col.logger.Error("Asynchronous error received, terminating process", zap.Error(err))
	case s := <-col.signalsChannel:
		col.logger.Info("Received signal from OS", zap.String("signal", s.String()))
	case <-col.shutdownChan:
		col.logger.Info("Received shutdown request")
	}
	col.stateChannel <- Closing
}

// setupConfigurationComponents loads the config and starts the components. If all the steps succeeds it
// sets the col.service with the service currently running.
func (col *Collector) setupConfigurationComponents(ctx context.Context) error {
	col.logger.Info("Loading configuration...")

	cp, err := col.parserProvider.Get()
	if err != nil {
		return fmt.Errorf("cannot load configuration's parser: %w", err)
	}

	cfg, err := col.configUnmarshaler.Unmarshal(cp, col.factories)
	if err != nil {
		return fmt.Errorf("cannot load configuration: %w", err)
	}

	if err = cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	col.logger.Info("Applying configuration...")

	service, err := newService(&svcSettings{
		BuildInfo:           col.info,
		Factories:           col.factories,
		Config:              cfg,
		Logger:              col.logger,
		TracerProvider:      col.tracerProvider,
		ZPagesSpanProcessor: col.zPagesSpanProcessor,
		AsyncErrorChannel:   col.asyncErrorChannel,
	})
	if err != nil {
		return err
	}

	err = service.Start(ctx)
	if err != nil {
		return err
	}

	col.service = service

	// If provider is watchable start a goroutine watching for updates.
	if watchable, ok := col.parserProvider.(parserprovider.Watchable); ok {
		go func() {
			err := watchable.WatchForUpdate()
			switch {
			// TODO: Move configsource.ErrSessionClosed to providerparser package to avoid depending on configsource.
			case errors.Is(err, configsource.ErrSessionClosed):
				// This is the case of shutdown of the whole collector server, nothing to do.
				col.logger.Info("Config WatchForUpdate closed", zap.Error(err))
				return
			default:
				col.logger.Warn("Config WatchForUpdated exited", zap.Error(err))
				if err := col.reloadService(context.Background()); err != nil {
					col.asyncErrorChannel <- err
				}
			}
		}()
	}

	return nil
}

func (col *Collector) execute(ctx context.Context) error {
	col.logger.Info("Starting "+col.info.Command+"...",
		zap.String("Version", col.info.Version),
		zap.Int("NumCPU", runtime.NumCPU()),
	)
	col.stateChannel <- Starting

	//  Add `mem-ballast-size-mib` warning message if it is still enabled
	//  TODO: will remove all `mem-ballast-size-mib` footprints after some baking time.
	if builder.MemBallastSize() > 0 {
		col.logger.Warn("`mem-ballast-size-mib` command line option has been deprecated. Please use `ballast extension` instead!")
	}

	col.asyncErrorChannel = make(chan error)

	err := col.setupConfigurationComponents(ctx)
	if err != nil {
		return err
	}

	// Get ballastSizeBytes if ballast extension is enabled and setup Telemetry.
	err = col.setupTelemetry(col.getBallastSize())
	if err != nil {
		return err
	}

	// Everything is ready, now run until an event requiring shutdown happens.
	col.runAndWaitForShutdownEvent()

	// Accumulate errors and proceed with shutting down remaining components.
	var errs []error

	// Begin shutdown sequence.
	col.logger.Info("Starting shutdown...")

	if closable, ok := col.parserProvider.(parserprovider.Closeable); ok {
		if err := closable.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to close config: %w", err))
		}
	}

	if col.service != nil {
		if err := col.service.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown service: %w", err))
		}
	}

	if err := collectorTelemetry.shutdown(); err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown collector telemetry: %w", err))
	}

	col.logger.Info("Shutdown complete.")
	col.stateChannel <- Closed
	close(col.stateChannel)

	return consumererror.Combine(errs)
}

// reloadService shutdowns the current col.service and setups a new one according
// to the latest configuration. It requires that col.parserProvider and col.factories
// are properly populated to finish successfully.
func (col *Collector) reloadService(ctx context.Context) error {
	if closeable, ok := col.parserProvider.(parserprovider.Closeable); ok {
		if err := closeable.Close(ctx); err != nil {
			return fmt.Errorf("failed close current config provider: %w", err)
		}
	}

	if col.service != nil {
		retiringService := col.service
		col.service = nil
		if err := retiringService.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown the retiring config: %w", err)
		}
	}

	if err := col.setupConfigurationComponents(ctx); err != nil {
		return fmt.Errorf("failed to setup configuration components: %w", err)
	}

	return nil
}

func (col *Collector) getBallastSize() uint64 {
	var ballastSize uint64
	extensions := col.service.GetExtensions()
	for _, extension := range extensions {
		if ext, ok := extension.(*ballastextension.MemoryBallast); ok {
			ballastSize = ext.GetBallastSize()
			break
		}
	}
	return ballastSize
}
