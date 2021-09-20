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
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"go.opentelemetry.io/contrib/zpages"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configunmarshaler"
	"go.opentelemetry.io/collector/config/experimental/configsource"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/extension/ballastextension"
	"go.opentelemetry.io/collector/service/internal"
	"go.opentelemetry.io/collector/service/internal/telemetrylogs"
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
// - Run starts the collector.
// - Run calls setupConfigurationComponents to handle configuration.
//   If configuration parser fails, collector's config can be reloaded.
//   Collector can be shutdown if parser gets a shutdown error.
// - Run runs runAndWaitForShutdownEvent and waits for a shutdown event.
//   SIGINT and SIGTERM, errors, and (*Collector).Shutdown can trigger the shutdown events.
// - Upon shutdown, pipelines are notified, then pipelines and extensions are shut down.
// - Users can call (*Collector).Shutdown anytime to shut down the collector.

// Collector represents a server providing the OpenTelemetry Collector service.
type Collector struct {
	set    CollectorSettings
	logger *zap.Logger

	tracerProvider      trace.TracerProvider
	meterProvider       metric.MeterProvider
	zPagesSpanProcessor *zpages.SpanProcessor

	service      *service
	stateChannel chan State

	// shutdownChan is used to terminate the collector.
	shutdownChan chan struct{}

	// signalsChannel is used to receive termination signals from the OS.
	signalsChannel chan os.Signal

	// asyncErrorChannel is used to signal a fatal error from any component.
	asyncErrorChannel chan error
}

// New creates and returns a new instance of Collector.
func New(set CollectorSettings) (*Collector, error) {
	if err := configcheck.ValidateConfigFromFactories(set.Factories); err != nil {
		return nil, err
	}

	if set.ParserProvider == nil {
		// use default provider.
		set.ParserProvider = parserprovider.Default()
	}

	if set.ConfigUnmarshaler == nil {
		// use default unmarshaler.
		set.ConfigUnmarshaler = configunmarshaler.NewDefault()
	}

	return &Collector{
		set:          set,
		stateChannel: make(chan State, Closed+1),
	}, nil
}

// GetStateChannel returns state channel of the collector server.
func (col *Collector) GetStateChannel() chan State {
	return col.stateChannel
}

// GetLogger returns logger used by the Collector.
// The logger is initialized after collector server start.
func (col *Collector) GetLogger() *zap.Logger {
	return col.logger
}

// Shutdown shuts down the collector server.
func (col *Collector) Shutdown() {
	defer func() {
		if r := recover(); r != nil {
			col.logger.Info("shutdownChan already closed")
		}
	}()
	close(col.shutdownChan)
}

// runAndWaitForShutdownEvent waits for one of the shutdown events that can happen.
func (col *Collector) runAndWaitForShutdownEvent() {
	col.logger.Info("Everything is ready. Begin running and processing data.")

	col.signalsChannel = make(chan os.Signal, 1)
	// Only notify with SIGTERM and SIGINT if graceful shutdown is enabled.
	if !col.set.DisableGracefulShutdown {
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
	cp, err := col.set.ParserProvider.Get(ctx)
	if err != nil {
		return fmt.Errorf("cannot load configuration's parser: %w", err)
	}

	cfg, err := col.set.ConfigUnmarshaler.Unmarshal(cp, col.set.Factories)
	if err != nil {
		return fmt.Errorf("cannot load configuration: %w", err)
	}

	if err = cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	if col.logger, err = telemetrylogs.NewLogger(cfg.Service.Telemetry.Logs, col.set.LoggingOptions); err != nil {
		return fmt.Errorf("failed to get logger: %w", err)
	}

	col.logger.Info("Applying configuration...")

	col.service, err = newService(&svcSettings{
		BuildInfo:           col.set.BuildInfo,
		Factories:           col.set.Factories,
		Config:              cfg,
		Logger:              col.logger,
		TracerProvider:      col.tracerProvider,
		MeterProvider:       col.meterProvider,
		ZPagesSpanProcessor: col.zPagesSpanProcessor,
		AsyncErrorChannel:   col.asyncErrorChannel,
	})
	if err != nil {
		return err
	}

	if err = col.service.Start(ctx); err != nil {
		return err
	}

	// If provider is watchable start a goroutine watching for updates.
	if watchable, ok := col.set.ParserProvider.(parserprovider.Watchable); ok {
		go col.watchForConfigUpdates(watchable)
	}

	return nil
}

// Run starts the collector according to the given configuration given, and waits for it to complete.
// Consecutive calls to Run are not allowed, Run shouldn't be called once a collector is shut down.
func (col *Collector) Run(ctx context.Context) error {
	col.zPagesSpanProcessor = zpages.NewSpanProcessor()
	col.tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(internal.AlwaysRecord()),
		sdktrace.WithSpanProcessor(col.zPagesSpanProcessor))

	// Set the constructed tracer provider as Global, in case any component uses the
	// global TracerProvider.
	otel.SetTracerProvider(col.tracerProvider)

	col.meterProvider = metric.NoopMeterProvider{}

	col.stateChannel <- Starting

	col.asyncErrorChannel = make(chan error)

	if err := col.setupConfigurationComponents(ctx); err != nil {
		return err
	}

	if err := collectorTelemetry.init(col.asyncErrorChannel, getBallastSize(col.service), col.logger); err != nil {
		return err
	}

	col.logger.Info("Starting "+col.set.BuildInfo.Command+"...",
		zap.String("Version", col.set.BuildInfo.Version),
		zap.Int("NumCPU", runtime.NumCPU()),
	)

	// Everything is ready, now run until an event requiring shutdown happens.
	col.runAndWaitForShutdownEvent()

	// Accumulate errors and proceed with shutting down remaining components.
	var errs []error

	// Begin shutdown sequence.
	col.logger.Info("Starting shutdown...")

	if err := col.set.ParserProvider.Close(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to close config: %w", err))
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
	if err := col.set.ParserProvider.Close(ctx); err != nil {
		return fmt.Errorf("failed close current config provider: %w", err)
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

func (col *Collector) watchForConfigUpdates(watchable parserprovider.Watchable) {
	err := watchable.WatchForUpdate()
	if errors.Is(err, configsource.ErrSessionClosed) {
		// This is the case of shutdown of the whole collector server, nothing to do.
		col.logger.Info("Config WatchForUpdate closed", zap.Error(err))
		return
	}
	col.logger.Warn("Config WatchForUpdated exited", zap.Error(err))
	if err = col.reloadService(context.Background()); err != nil {
		col.asyncErrorChannel <- err
	}
}

func getBallastSize(host component.Host) uint64 {
	var ballastSize uint64
	extensions := host.GetExtensions()
	for _, extension := range extensions {
		if ext, ok := extension.(*ballastextension.MemoryBallast); ok {
			ballastSize = ext.GetBallastSize()
			break
		}
	}
	return ballastSize
}
