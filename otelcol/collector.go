// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelcol handles the command-line, configuration, and runs the OpenTelemetry Collector.
// It contains the main [Collector] struct and its constructor [NewCollector].
// [Collector.Run] starts the Collector and then blocks until it shuts down.
package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/otelcol/internal/grpclog"
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
	// Factories returns component factories for the collector.
	//
	// TODO(13263) This is a dangerous "bare" function value, should define an interface
	// following style guidelines.
	Factories func() (Factories, error)

	// BuildInfo provides collector start information.
	BuildInfo component.BuildInfo

	// DisableGracefulShutdown disables the automatic graceful shutdown
	// of the collector on SIGINT or SIGTERM.
	// Users who want to handle signals themselves can disable this behavior
	// and manually handle the signals to shutdown the collector.
	DisableGracefulShutdown bool

	// ConfigProviderSettings allows configuring the way the Collector retrieves its configuration
	// The Collector will reload based on configuration changes from the ConfigProvider if any
	// confmap.Providers watch for configuration changes.
	ConfigProviderSettings ConfigProviderSettings

	// ProviderModules maps provider schemes to their respective go modules.
	ProviderModules map[string]string

	// ConverterModules maps converter names to their respective go modules.
	ConverterModules []string

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
	set            CollectorSettings
	buildZapLogger func(zap.Config, ...zap.Option) (*zap.Logger, error)

	configProvider *ConfigProvider

	serviceConfig *service.Config
	service       *service.Service
	state         *atomic.Int64

	// shutdownChan is used to terminate the collector.
	shutdownChan chan struct{}
	shutdownOnce sync.Once

	// signalsChannel is used to receive termination signals from the OS.
	signalsChannel chan os.Signal
	// asyncErrorChannel is used to signal a fatal error from any component.
	asyncErrorChannel          chan error
	bc                         *bufferedCore
	updateConfigProviderLogger func(core zapcore.Core)
}

// NewCollector creates and returns a new instance of Collector.
func NewCollector(set CollectorSettings) (*Collector, error) {
	bc := newBufferedCore(zapcore.DebugLevel)
	cc := newCollectorCore(bc)
	options := append([]zap.Option{zap.WithCaller(true)}, set.LoggingOptions...)
	logger := zap.New(cc, options...)
	set.ConfigProviderSettings.ResolverSettings.ProviderSettings = confmap.ProviderSettings{Logger: logger}
	set.ConfigProviderSettings.ResolverSettings.ConverterSettings = confmap.ConverterSettings{Logger: logger}

	configProvider, err := NewConfigProvider(set.ConfigProviderSettings)
	if err != nil {
		return nil, err
	}

	state := new(atomic.Int64)
	state.Store(int64(StateStarting))
	return &Collector{
		set:            set,
		buildZapLogger: zap.Config.Build,
		state:          state,
		shutdownChan:   make(chan struct{}),
		// Per signal.Notify documentation, a size of the channel equaled with
		// the number of signals getting notified on is recommended.
		signalsChannel:             make(chan os.Signal, 3),
		asyncErrorChannel:          make(chan error),
		configProvider:             configProvider,
		bc:                         bc,
		updateConfigProviderLogger: cc.SetCore,
	}, nil
}

// GetState returns current state of the collector server.
func (col *Collector) GetState() State {
	return State(col.state.Load())
}

// Shutdown shuts down the collector server.
func (col *Collector) Shutdown() {
	col.shutdownOnce.Do(func() {
		close(col.shutdownChan)
	})
}

func buildModuleInfo(m map[component.Type]string) map[component.Type]service.ModuleInfo {
	moduleInfo := make(map[component.Type]service.ModuleInfo)
	for k, v := range m {
		moduleInfo[k] = service.ModuleInfo{BuilderRef: v}
	}
	return moduleInfo
}

// setupConfigurationComponents loads the config, creates the graph, and starts the components. If all the steps succeeds it
// sets the col.service with the service currently running.
func (col *Collector) setupConfigurationComponents(ctx context.Context) error {
	col.setCollectorState(StateStarting)

	factories, err := col.set.Factories()
	if err != nil {
		return fmt.Errorf("failed to initialize factories: %w", err)
	}

	cfg, err := col.configProvider.Get(ctx, factories)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if err = xconfmap.Validate(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	col.serviceConfig = &cfg.Service

	conf := confmap.New()

	if err = conf.Marshal(cfg); err != nil {
		return fmt.Errorf("could not marshal configuration: %w", err)
	}

	// Wrap the buildZapLogger to append LoggingOptions from collector settings,
	// since service.Settings.LoggingOptions is deprecated.
	buildZapLogger := col.buildZapLogger
	if len(col.set.LoggingOptions) > 0 {
		origBuildZapLogger := buildZapLogger
		buildZapLogger = func(zapCfg zap.Config, opts ...zap.Option) (*zap.Logger, error) {
			opts = append(opts, col.set.LoggingOptions...)
			return origBuildZapLogger(zapCfg, opts...)
		}
	}

	col.service, err = service.New(ctx, service.Settings{
		BuildInfo:     col.set.BuildInfo,
		CollectorConf: conf,

		ReceiversConfigs:    cfg.Receivers,
		ReceiversFactories:  factories.Receivers,
		ProcessorsConfigs:   cfg.Processors,
		ProcessorsFactories: factories.Processors,
		ExportersConfigs:    cfg.Exporters,
		ExportersFactories:  factories.Exporters,
		ConnectorsConfigs:   cfg.Connectors,
		ConnectorsFactories: factories.Connectors,
		ExtensionsConfigs:   cfg.Extensions,
		ExtensionsFactories: factories.Extensions,

		ModuleInfos: service.ModuleInfos{
			Receiver:  buildModuleInfo(factories.ReceiverModules),
			Processor: buildModuleInfo(factories.ProcessorModules),
			Exporter:  buildModuleInfo(factories.ExporterModules),
			Extension: buildModuleInfo(factories.ExtensionModules),
			Connector: buildModuleInfo(factories.ConnectorModules),
		},
		AsyncErrorChannel: col.asyncErrorChannel,
		BuildZapLogger:    buildZapLogger,
		TelemetryFactory:  factories.Telemetry,
	}, cfg.Service)
	if err != nil {
		return err
	}
	if col.updateConfigProviderLogger != nil {
		col.updateConfigProviderLogger(col.service.Logger().Core())
	}
	if col.bc != nil {
		x := col.bc.TakeLogs()
		for _, log := range x {
			ce := col.service.Logger().Core().Check(log.Entry, nil)
			if ce != nil {
				ce.Write(log.Context...)
			}
		}
	}

	if !col.set.SkipSettingGRPCLogger {
		grpclog.SetLogger(col.service.Logger())
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

func (col *Collector) DryRun(ctx context.Context) error {
	factories, err := col.set.Factories()
	if err != nil {
		return fmt.Errorf("failed to initialize factories: %w", err)
	}

	cfg, err := col.configProvider.Get(ctx, factories)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if err := xconfmap.Validate(cfg); err != nil {
		return err
	}

	return service.Validate(ctx, service.Settings{
		BuildInfo:           col.set.BuildInfo,
		ReceiversConfigs:    cfg.Receivers,
		ReceiversFactories:  factories.Receivers,
		ProcessorsConfigs:   cfg.Processors,
		ProcessorsFactories: factories.Processors,
		ExportersConfigs:    cfg.Exporters,
		ExportersFactories:  factories.Exporters,
		ConnectorsConfigs:   cfg.Connectors,
		ConnectorsFactories: factories.Connectors,
		TelemetryFactory:    factories.Telemetry,
	}, service.Config{
		Pipelines: cfg.Service.Pipelines,
	})
}

func newFallbackLogger(options []zap.Option) (*zap.Logger, error) {
	ec := zap.NewProductionEncoderConfig()
	ec.EncodeTime = zapcore.ISO8601TimeEncoder
	zapCfg := &zap.Config{
		Level:            zap.NewAtomicLevelAt(zapcore.DebugLevel),
		Encoding:         "console",
		EncoderConfig:    ec,
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	return zapCfg.Build(options...)
}

// Run starts the collector according to the given configuration, and waits for it to complete.
// Consecutive calls to Run are not allowed, Run shouldn't be called once a collector is shut down.
// Sets up the control logic for config reloading and shutdown.
func (col *Collector) Run(ctx context.Context) error {
	// setupConfigurationComponents is the "main" function responsible for startup
	if err := col.setupConfigurationComponents(ctx); err != nil {
		col.setCollectorState(StateClosed)
		logger, loggerErr := newFallbackLogger(col.set.LoggingOptions)
		if loggerErr != nil {
			return errors.Join(err, fmt.Errorf("unable to create fallback logger: %w", loggerErr))
		}

		if col.bc != nil {
			x := col.bc.TakeLogs()
			for _, log := range x {
				ce := logger.Core().Check(log.Entry, nil)
				if ce != nil {
					ce.Write(log.Context...)
				}
			}
		}

		return err
	}

	// Always notify with SIGHUP for configuration reloading.
	signal.Notify(col.signalsChannel, syscall.SIGHUP)
	defer signal.Stop(col.signalsChannel)

	// Only notify with SIGTERM and SIGINT if graceful shutdown is enabled.
	if !col.set.DisableGracefulShutdown {
		signal.Notify(col.signalsChannel, os.Interrupt, syscall.SIGTERM)
	}

	// Control loop: selects between channels for various interrupts - when this loop is broken, the collector exits.
	// If a configuration reload fails, we return without waiting for graceful shutdown.
LOOP:
	for {
		select {
		case err := <-col.configProvider.Watch():
			if err != nil {
				col.service.Logger().Error("Config watch failed", zap.Error(err))
				break LOOP
			}
			if err := col.reloadConfiguration(ctx); err != nil {
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
			return col.shutdown(context.Background()) //nolint:contextcheck
		}
	}
	return col.shutdown(ctx)
}

func (col *Collector) shutdown(ctx context.Context) error {
	col.setCollectorState(StateClosing)

	// Accumulate errors and proceed with shutting down remaining components.
	var errs error

	if err := col.configProvider.Shutdown(ctx); err != nil {
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
	col.state.Store(int64(state))
}
