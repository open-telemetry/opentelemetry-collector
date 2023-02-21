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
package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/atomic"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/otelcol/internal/grpclog"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service"
)

// State defines Collector's state.
type State int

const (
	StateStarting State = iota
	StateRunning
	StateClosing
	StateClosed
)

func (s State) String() string {
	switch s {
	case StateStarting:
		return "Starting"
	case StateRunning:
		return "Running"
	case StateClosing:
		return "Closing"
	case StateClosed:
		return "Closed"
	}
	return "UNKNOWN"
}

// CollectorSettings holds configuration for creating a new Collector.
type CollectorSettings struct {
	// Factories service factories.
	Factories Factories

	// BuildInfo provides collector start information.
	BuildInfo component.BuildInfo

	// DisableGracefulShutdown disables the automatic graceful shutdown
	// of the collector on SIGINT or SIGTERM.
	// Users who want to handle signals themselves can disable this behavior
	// and manually handle the signals to shutdown the collector.
	DisableGracefulShutdown bool

	// ConfigProvider provides the service configuration.
	// If the provider watches for configuration change, collector may reload the new configuration upon changes.
	ConfigProvider ConfigProvider

	// LoggingOptions provides a way to change behavior of zap logging.
	LoggingOptions []zap.Option

	// SkipSettingGRPCLogger avoids setting the grpc logger
	SkipSettingGRPCLogger bool
}

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
	set CollectorSettings

	service *service.Service
	state   *atomic.Int32

	// shutdownChan is used to terminate the collector.
	shutdownChan chan struct{}
	// signalsChannel is used to receive termination signals from the OS.
	signalsChannel chan os.Signal
	// asyncErrorChannel is used to signal a fatal error from any component.
	asyncErrorChannel chan error
}

// NewCollector creates and returns a new instance of Collector.
func NewCollector(set CollectorSettings) (*Collector, error) {
	if set.ConfigProvider == nil {
		return nil, errors.New("invalid nil config provider")
	}

	return &Collector{
		set:          set,
		state:        atomic.NewInt32(int32(StateStarting)),
		shutdownChan: make(chan struct{}),
		// Per signal.Notify documentation, a size of the channel equaled with
		// the number of signals getting notified on is recommended.
		signalsChannel:    make(chan os.Signal, 3),
		asyncErrorChannel: make(chan error),
	}, nil
}

// GetState returns current state of the collector server.
func (col *Collector) GetState() State {
	return State(col.state.Load())
}

// Shutdown shuts down the collector server.
func (col *Collector) Shutdown() {
	// Only shutdown if we're in a Running or Starting State else noop
	state := col.GetState()
	if state == StateRunning || state == StateStarting {
		defer func() {
			recover() // nolint:errcheck
		}()
		close(col.shutdownChan)
	}
}

// setupConfigurationComponents loads the config and starts the components. If all the steps succeeds it
// sets the col.service with the service currently running.
func (col *Collector) setupConfigurationComponents(ctx context.Context) error {
	col.setCollectorState(StateStarting)

	cfg, err := col.set.ConfigProvider.Get(ctx, col.set.Factories)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if err = cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	col.service, err = service.New(ctx, service.Settings{
		BuildInfo:         col.set.BuildInfo,
		Receivers:         receiver.NewBuilder(cfg.Receivers, col.set.Factories.Receivers),
		Processors:        processor.NewBuilder(cfg.Processors, col.set.Factories.Processors),
		Exporters:         exporter.NewBuilder(cfg.Exporters, col.set.Factories.Exporters),
		Connectors:        connector.NewBuilder(cfg.Connectors, col.set.Factories.Connectors),
		Extensions:        extension.NewBuilder(cfg.Extensions, col.set.Factories.Extensions),
		AsyncErrorChannel: col.asyncErrorChannel,
		LoggingOptions:    col.set.LoggingOptions,
	}, cfg.Service)
	if err != nil {
		return err
	}

	if !col.set.SkipSettingGRPCLogger {
		grpclog.SetLogger(col.service.Logger(), cfg.Service.Telemetry.Logs.Level)
	}

	if err = col.service.Start(ctx); err != nil {
		return multierr.Combine(err, col.service.Shutdown(ctx))
	}
	col.setCollectorState(StateRunning)
	return nil
}

func (col *Collector) reloadConfiguration(ctx context.Context) error {
	col.service.Logger().Warn("Config updated, restart service")
	col.setCollectorState(StateClosing)

	if err := col.service.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown the retiring config: %w", err)
	}

	if err := col.setupConfigurationComponents(ctx); err != nil {
		return fmt.Errorf("failed to setup configuration components: %w", err)
	}

	return nil
}

// Run starts the collector according to the given configuration, and waits for it to complete.
// Consecutive calls to Run are not allowed, Run shouldn't be called once a collector is shut down.
func (col *Collector) Run(ctx context.Context) error {
	if err := col.setupConfigurationComponents(ctx); err != nil {
		col.setCollectorState(StateClosed)
		return err
	}

	// Always notify with SIGHUP for configuration reloading.
	signal.Notify(col.signalsChannel, syscall.SIGHUP)
	defer signal.Stop(col.signalsChannel)

	// Only notify with SIGTERM and SIGINT if graceful shutdown is enabled.
	if !col.set.DisableGracefulShutdown {
		signal.Notify(col.signalsChannel, os.Interrupt, syscall.SIGTERM)
	}

LOOP:
	for {
		select {
		case err := <-col.set.ConfigProvider.Watch():
			if err != nil {
				col.service.Logger().Error("Config watch failed", zap.Error(err))
				break LOOP
			}
			if err = col.reloadConfiguration(ctx); err != nil {
				return err
			}
		case err := <-col.asyncErrorChannel:
			col.service.Logger().Error("Asynchronous error received, terminating process", zap.Error(err))
			break LOOP
		case s := <-col.signalsChannel:
			col.service.Logger().Info("Received signal from OS", zap.String("signal", s.String()))
			if s != syscall.SIGHUP {
				break LOOP
			}
			if err := col.reloadConfiguration(ctx); err != nil {
				return err
			}
		case <-col.shutdownChan:
			col.service.Logger().Info("Received shutdown request")
			break LOOP
		case <-ctx.Done():
			col.service.Logger().Info("Context done, terminating process", zap.Error(ctx.Err()))
			// Call shutdown with background context as the passed in context has been canceled
			return col.shutdown(context.Background())
		}
	}
	return col.shutdown(ctx)
}

func (col *Collector) shutdown(ctx context.Context) error {
	col.setCollectorState(StateClosing)

	// Accumulate errors and proceed with shutting down remaining components.
	var errs error

	if err := col.set.ConfigProvider.Shutdown(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown config provider: %w", err))
	}

	// shutdown service
	if err := col.service.Shutdown(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown service after error: %w", err))
	}

	col.setCollectorState(StateClosed)

	return errs
}

// setCollectorState provides current state of the collector
func (col *Collector) setCollectorState(state State) {
	col.state.Store(int32(state))
}
