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

package service // import "go.opentelemetry.io/collector/service"

import (
	"context"
	"fmt"
	"runtime"

	"go.opentelemetry.io/otel/metric"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/internal/proctelemetry"
	"go.opentelemetry.io/collector/service/telemetry"
)

// Settings holds configuration for building a new service.
type Settings struct {
	// BuildInfo provides collector start information.
	BuildInfo component.BuildInfo

	// Receivers builder for receivers.
	Receivers *receiver.Builder

	// Processors builder for processors.
	Processors *processor.Builder

	// Exporters builder for exporters.
	Exporters *exporter.Builder

	// Connectors builder for connectors.
	Connectors *connector.Builder

	// Extensions builder for extensions.
	Extensions *extension.Builder

	// AsyncErrorChannel is the channel that is used to report fatal errors.
	AsyncErrorChannel chan error

	// LoggingOptions provides a way to change behavior of zap logging.
	LoggingOptions []zap.Option

	// For testing purpose only.
	registry *featuregate.Registry
}

// Service represents the implementation of a component.Host.
type Service struct {
	buildInfo            component.BuildInfo
	telemetry            *telemetry.Telemetry
	telemetrySettings    component.TelemetrySettings
	host                 *serviceHost
	telemetryInitializer *telemetryInitializer
}

func New(ctx context.Context, set Settings, cfg Config) (*Service, error) {
	reg := set.registry
	if reg == nil {
		reg = featuregate.GetRegistry()
	}
	srv := &Service{
		buildInfo: set.BuildInfo,
		host: &serviceHost{
			receivers:         set.Receivers,
			processors:        set.Processors,
			exporters:         set.Exporters,
			connectors:        set.Connectors,
			extensions:        set.Extensions,
			buildInfo:         set.BuildInfo,
			asyncErrorChannel: set.AsyncErrorChannel,
		},
		telemetryInitializer: newColTelemetry(reg),
	}
	var err error
	srv.telemetry, err = telemetry.New(ctx, telemetry.Settings{ZapOptions: set.LoggingOptions}, cfg.Telemetry)
	if err != nil {
		return nil, fmt.Errorf("failed to get logger: %w", err)
	}
	srv.telemetrySettings = component.TelemetrySettings{
		Logger:         srv.telemetry.Logger(),
		TracerProvider: srv.telemetry.TracerProvider(),
		MeterProvider:  metric.NewNoopMeterProvider(),
		MetricsLevel:   cfg.Telemetry.Metrics.Level,
	}

	if err = srv.telemetryInitializer.init(set.BuildInfo, srv.telemetrySettings.Logger, cfg.Telemetry, set.AsyncErrorChannel); err != nil {
		return nil, fmt.Errorf("failed to initialize telemetry: %w", err)
	}
	srv.telemetrySettings.MeterProvider = srv.telemetryInitializer.mp

	// process the configuration and initialize the pipeline
	if err = srv.initExtensionsAndPipeline(ctx, set, cfg); err != nil {
		// If pipeline initialization fails then shut down the telemetry server
		if shutdownErr := srv.telemetryInitializer.shutdown(); shutdownErr != nil {
			err = multierr.Append(err, fmt.Errorf("failed to shutdown collector telemetry: %w", shutdownErr))
		}

		return nil, err
	}

	return srv, nil
}

// Start starts the extensions and pipelines. If Start fails Shutdown should be called to ensure a clean state.
func (srv *Service) Start(ctx context.Context) error {
	srv.telemetrySettings.Logger.Info("Starting "+srv.buildInfo.Command+"...",
		zap.String("Version", srv.buildInfo.Version),
		zap.Int("NumCPU", runtime.NumCPU()),
	)

	if err := srv.host.serviceExtensions.Start(ctx, srv.host); err != nil {
		return fmt.Errorf("failed to start extensions: %w", err)
	}

	if err := srv.host.pipelines.StartAll(ctx, srv.host); err != nil {
		return fmt.Errorf("cannot start pipelines: %w", err)
	}

	if err := srv.host.serviceExtensions.NotifyPipelineReady(); err != nil {
		return err
	}

	srv.telemetrySettings.Logger.Info("Everything is ready. Begin running and processing data.")
	return nil
}

func (srv *Service) Shutdown(ctx context.Context) error {
	// Accumulate errors and proceed with shutting down remaining components.
	var errs error

	// Begin shutdown sequence.
	srv.telemetrySettings.Logger.Info("Starting shutdown...")

	if err := srv.host.serviceExtensions.NotifyPipelineNotReady(); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to notify that pipeline is not ready: %w", err))
	}

	if err := srv.host.pipelines.ShutdownAll(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown pipelines: %w", err))
	}

	if err := srv.host.serviceExtensions.Shutdown(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown extensions: %w", err))
	}

	srv.telemetrySettings.Logger.Info("Shutdown complete.")

	if err := srv.telemetry.Shutdown(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown telemetry: %w", err))
	}

	if err := srv.telemetryInitializer.shutdown(); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown collector telemetry: %w", err))
	}
	return errs
}

func (srv *Service) initExtensionsAndPipeline(ctx context.Context, set Settings, cfg Config) error {
	var err error
	extensionsSettings := extensions.Settings{
		Telemetry:  srv.telemetrySettings,
		BuildInfo:  srv.buildInfo,
		Extensions: srv.host.extensions,
	}
	if srv.host.serviceExtensions, err = extensions.New(ctx, extensionsSettings, cfg.Extensions); err != nil {
		return fmt.Errorf("failed build extensions: %w", err)
	}

	pSet := pipelinesSettings{
		Telemetry:       srv.telemetrySettings,
		BuildInfo:       srv.buildInfo,
		Receivers:       set.Receivers,
		Processors:      set.Processors,
		Exporters:       set.Exporters,
		PipelineConfigs: cfg.Pipelines,
	}
	if srv.host.pipelines, err = buildPipelines(ctx, pSet); err != nil {
		return fmt.Errorf("cannot build pipelines: %w", err)
	}

	if cfg.Telemetry.Metrics.Level != configtelemetry.LevelNone && cfg.Telemetry.Metrics.Address != "" {
		// The process telemetry initialization requires the ballast size, which is available after the extensions are initialized.
		if err = proctelemetry.RegisterProcessMetrics(srv.telemetryInitializer.ocRegistry, getBallastSize(srv.host)); err != nil {
			return fmt.Errorf("failed to register process metrics: %w", err)
		}
	}

	return nil
}

// Logger returns the logger created for this service.
// This is a temporary API that may be removed soon after investigating how the collector should record different events.
func (srv *Service) Logger() *zap.Logger {
	return srv.telemetrySettings.Logger
}

func getBallastSize(host component.Host) uint64 {
	for _, ext := range host.GetExtensions() {
		if bExt, ok := ext.(interface{ GetBallastSize() uint64 }); ok {
			return bExt.GetBallastSize()
		}
	}
	return 0
}
