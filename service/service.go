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
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/internal/proctelemetry"
	"go.opentelemetry.io/collector/service/telemetry"
)

// Settings holds configuration for building a new service.
type Settings struct {
	// Factories component factories.
	Factories component.Factories

	// BuildInfo provides collector start information.
	BuildInfo component.BuildInfo

	// Config represents the configuration of the service.
	Config *Config

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
	config               *Config
	telemetry            *telemetry.Telemetry
	telemetrySettings    component.TelemetrySettings
	host                 *serviceHost
	telemetryInitializer *telemetryInitializer
}

func New(set Settings) (*Service, error) {
	reg := set.registry
	if reg == nil {
		reg = featuregate.GetRegistry()
	}
	srv := &Service{
		buildInfo: set.BuildInfo,
		config:    set.Config,
		host: &serviceHost{
			factories:         set.Factories,
			buildInfo:         set.BuildInfo,
			asyncErrorChannel: set.AsyncErrorChannel,
		},
		telemetryInitializer: newColTelemetry(reg),
	}
	var err error
	srv.telemetry, err = telemetry.New(context.Background(), telemetry.Settings{ZapOptions: set.LoggingOptions}, set.Config.Service.Telemetry)
	if err != nil {
		return nil, fmt.Errorf("failed to get logger: %w", err)
	}
	srv.telemetrySettings = component.TelemetrySettings{
		Logger:         srv.telemetry.Logger(),
		TracerProvider: srv.telemetry.TracerProvider(),
		MeterProvider:  metric.NewNoopMeterProvider(),
		MetricsLevel:   set.Config.Service.Telemetry.Metrics.Level,
	}

	if err = srv.telemetryInitializer.init(set.BuildInfo, srv.telemetrySettings.Logger, set.Config.Service.Telemetry, set.AsyncErrorChannel); err != nil {
		return nil, fmt.Errorf("failed to initialize telemetry: %w", err)
	}
	srv.telemetrySettings.MeterProvider = srv.telemetryInitializer.mp

	// process the configuration and initialize the pipeline
	if err = srv.initExtensionsAndPipeline(set); err != nil {
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

	if err := srv.host.extensions.Start(ctx, srv.host); err != nil {
		return fmt.Errorf("failed to start extensions: %w", err)
	}

	if err := srv.host.pipelines.StartAll(ctx, srv.host); err != nil {
		return fmt.Errorf("cannot start pipelines: %w", err)
	}

	if err := srv.host.extensions.NotifyPipelineReady(); err != nil {
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

	if err := srv.host.extensions.NotifyPipelineNotReady(); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to notify that pipeline is not ready: %w", err))
	}

	if err := srv.host.pipelines.ShutdownAll(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown pipelines: %w", err))
	}

	if err := srv.host.extensions.Shutdown(ctx); err != nil {
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

func (srv *Service) initExtensionsAndPipeline(set Settings) error {
	var err error
	extensionsSettings := extensions.Settings{
		Telemetry: srv.telemetrySettings,
		BuildInfo: srv.buildInfo,
		Configs:   srv.config.Extensions,
		Factories: srv.host.factories.Extensions,
	}
	if srv.host.extensions, err = extensions.New(context.Background(), extensionsSettings, srv.config.Service.Extensions); err != nil {
		return fmt.Errorf("failed build extensions: %w", err)
	}

	pSet := pipelinesSettings{
		Telemetry:          srv.telemetrySettings,
		BuildInfo:          srv.buildInfo,
		ReceiverFactories:  srv.host.factories.Receivers,
		ReceiverConfigs:    srv.config.Receivers,
		ProcessorFactories: srv.host.factories.Processors,
		ProcessorConfigs:   srv.config.Processors,
		ExporterFactories:  srv.host.factories.Exporters,
		ExporterConfigs:    srv.config.Exporters,
		PipelineConfigs:    srv.config.Service.Pipelines,
	}
	if srv.host.pipelines, err = buildPipelines(context.Background(), pSet); err != nil {
		return fmt.Errorf("cannot build pipelines: %w", err)
	}

	if set.Config.Service.Telemetry.Metrics.Level != configtelemetry.LevelNone && set.Config.Service.Telemetry.Metrics.Address != "" {
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
	var ballastSize uint64
	extensions := host.GetExtensions()
	for _, extension := range extensions {
		if ext, ok := extension.(interface{ GetBallastSize() uint64 }); ok {
			ballastSize = ext.GetBallastSize()
			break
		}
	}
	return ballastSize
}
