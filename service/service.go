// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

package service // import "go.opentelemetry.io/collector/service"

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/graph"
	"go.opentelemetry.io/collector/service/internal/proctelemetry"
	"go.opentelemetry.io/collector/service/internal/resource"
	"go.opentelemetry.io/collector/service/internal/status"
	"go.opentelemetry.io/collector/service/pipelines"
	"go.opentelemetry.io/collector/service/telemetry"
)

// useOtelWithSDKConfigurationForInternalTelemetryFeatureGate is the feature gate that controls whether the collector
// supports configuring the OpenTelemetry SDK via configuration
var _ = featuregate.GlobalRegistry().MustRegister(
	"telemetry.useOtelWithSDKConfigurationForInternalTelemetry",
	featuregate.StageStable,
	featuregate.WithRegisterToVersion("v0.110.0"),
	featuregate.WithRegisterDescription("controls whether the collector supports extended OpenTelemetry"+
		"configuration for internal telemetry"))

// Settings holds configuration for building a new Service.
type Settings struct {
	// BuildInfo provides collector start information.
	BuildInfo component.BuildInfo

	// CollectorConf contains the Collector's current configuration
	CollectorConf *confmap.Conf

	// Receivers configuration to its builder.
	ReceiversConfigs   map[component.ID]component.Config
	ReceiversFactories map[component.Type]receiver.Factory

	// Processors configuration to its builder.
	ProcessorsConfigs   map[component.ID]component.Config
	ProcessorsFactories map[component.Type]processor.Factory

	// exporters configuration to its builder.
	ExportersConfigs   map[component.ID]component.Config
	ExportersFactories map[component.Type]exporter.Factory

	// Connectors configuration to its builder.
	ConnectorsConfigs   map[component.ID]component.Config
	ConnectorsFactories map[component.Type]connector.Factory

	// Extensions builder for extensions.
	Extensions builders.Extension

	// Extensions configuration to its builder.
	ExtensionsConfigs   map[component.ID]component.Config
	ExtensionsFactories map[component.Type]extension.Factory

	// ModuleInfo describes the go module for each component.
	ModuleInfo extension.ModuleInfo

	// AsyncErrorChannel is the channel that is used to report fatal errors.
	AsyncErrorChannel chan error

	// LoggingOptions provides a way to change behavior of zap logging.
	LoggingOptions []zap.Option
}

// Service represents the implementation of a component.Host.
type Service struct {
	buildInfo         component.BuildInfo
	telemetrySettings component.TelemetrySettings
	host              *graph.Host
	collectorConf     *confmap.Conf
}

// New creates a new Service, its telemetry, and Components.
func New(ctx context.Context, set Settings, cfg Config) (*Service, error) {
	srv := &Service{
		buildInfo: set.BuildInfo,
		host: &graph.Host{
			Receivers:  builders.NewReceiver(set.ReceiversConfigs, set.ReceiversFactories),
			Processors: builders.NewProcessor(set.ProcessorsConfigs, set.ProcessorsFactories),
			Exporters:  builders.NewExporter(set.ExportersConfigs, set.ExportersFactories),
			Connectors: builders.NewConnector(set.ConnectorsConfigs, set.ConnectorsFactories),
			Extensions: builders.NewExtension(set.ExtensionsConfigs, set.ExtensionsFactories),

			ModuleInfo:        set.ModuleInfo,
			BuildInfo:         set.BuildInfo,
			AsyncErrorChannel: set.AsyncErrorChannel,
		},
		collectorConf: set.CollectorConf,
	}

	// Fetch data for internal telemetry like instance id and sdk version to provide for internal telemetry.
	res := resource.New(set.BuildInfo, cfg.Telemetry.Resource)
	pcommonRes := pdataFromSdk(res)

	telFactory := telemetry.NewFactory()
	telset := telemetry.Settings{
		BuildInfo:  set.BuildInfo,
		ZapOptions: set.LoggingOptions,
	}

	logger, err := telFactory.CreateLogger(ctx, telset, &cfg.Telemetry)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	tracerProvider, err := telFactory.CreateTracerProvider(ctx, telset, &cfg.Telemetry)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer provider: %w", err)
	}

	logger.Info("Setting up own telemetry...")

	mp, err := telFactory.CreateMeterProvider(ctx, telset, &cfg.Telemetry)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric provider: %w", err)
	}

	logsAboutMeterProvider(logger, cfg.Telemetry.Metrics, mp)
	srv.telemetrySettings = component.TelemetrySettings{
		LeveledMeterProvider: func(level configtelemetry.Level) metric.MeterProvider {
			if level <= cfg.Telemetry.Metrics.Level {
				return mp
			}
			return noop.MeterProvider{}
		},
		Logger:         logger,
		MeterProvider:  mp,
		TracerProvider: tracerProvider,
		MetricsLevel:   cfg.Telemetry.Metrics.Level,
		// Construct telemetry attributes from build info and config's resource attributes.
		Resource: pcommonRes,
	}
	srv.host.Reporter = status.NewReporter(srv.host.NotifyComponentStatusChange, func(err error) {
		if errors.Is(err, status.ErrStatusNotReady) {
			logger.Warn("Invalid transition", zap.Error(err))
		}
		// ignore other errors as they represent invalid state transitions and are considered benign.
	})

	if err = srv.initGraph(ctx, cfg); err != nil {
		err = multierr.Append(err, srv.shutdownTelemetry(ctx))
		return nil, err
	}

	// process the configuration and initialize the pipeline
	if err = srv.initExtensions(ctx, cfg.Extensions); err != nil {
		err = multierr.Append(err, srv.shutdownTelemetry(ctx))
		return nil, err
	}

	if err = proctelemetry.RegisterProcessMetrics(srv.telemetrySettings); err != nil {
		return nil, fmt.Errorf("failed to register process metrics: %w", err)
	}

	return srv, nil
}

func logsAboutMeterProvider(logger *zap.Logger, cfg telemetry.MetricsConfig, mp metric.MeterProvider) {
	if cfg.Level == configtelemetry.LevelNone || len(cfg.Readers) == 0 {
		logger.Info("Skipped telemetry setup.")
		return
	}

	//nolint
	if len(cfg.Address) != 0 {
		logger.Warn("service::telemetry::metrics::address is being deprecated in favor of service::telemetry::metrics::readers")
	}

	if lmp, ok := mp.(interface {
		LogAboutServers(logger *zap.Logger, cfg telemetry.MetricsConfig)
	}); ok {
		lmp.LogAboutServers(logger, cfg)
	}
}

// Start starts the extensions and pipelines. If Start fails Shutdown should be called to ensure a clean state.
// Start does the following steps in order:
// 1. Start all extensions.
// 2. Notify extensions about Collector configuration
// 3. Start all pipelines.
// 4. Notify extensions that the pipeline is ready.
func (srv *Service) Start(ctx context.Context) error {
	srv.telemetrySettings.Logger.Info("Starting "+srv.buildInfo.Command+"...",
		zap.String("Version", srv.buildInfo.Version),
		zap.Int("NumCPU", runtime.NumCPU()),
	)

	// enable status reporting
	srv.host.Reporter.Ready()

	if err := srv.host.ServiceExtensions.Start(ctx, srv.host); err != nil {
		return fmt.Errorf("failed to start extensions: %w", err)
	}

	if srv.collectorConf != nil {
		if err := srv.host.ServiceExtensions.NotifyConfig(ctx, srv.collectorConf); err != nil {
			return err
		}
	}

	if err := srv.host.Pipelines.StartAll(ctx, srv.host); err != nil {
		return fmt.Errorf("cannot start pipelines: %w", err)
	}

	if err := srv.host.ServiceExtensions.NotifyPipelineReady(); err != nil {
		return err
	}

	srv.telemetrySettings.Logger.Info("Everything is ready. Begin running and processing data.")
	return nil
}

func (srv *Service) shutdownTelemetry(ctx context.Context) error {
	// The metric.MeterProvider and trace.TracerProvider interfaces do not have a Shutdown method.
	// To shutdown the providers we try to cast to this interface, which matches the type signature used in the SDK.
	type shutdownable interface {
		Shutdown(context.Context) error
	}

	var err error
	if prov, ok := srv.telemetrySettings.MeterProvider.(shutdownable); ok {
		if shutdownErr := prov.Shutdown(ctx); shutdownErr != nil {
			err = multierr.Append(err, fmt.Errorf("failed to shutdown meter provider: %w", shutdownErr))
		}
	}

	if prov, ok := srv.telemetrySettings.TracerProvider.(shutdownable); ok {
		if shutdownErr := prov.Shutdown(ctx); shutdownErr != nil {
			err = multierr.Append(err, fmt.Errorf("failed to shutdown tracer provider: %w", shutdownErr))
		}
	}
	return err
}

// Shutdown the service. Shutdown will do the following steps in order:
// 1. Notify extensions that the pipeline is shutting down.
// 2. Shutdown all pipelines.
// 3. Shutdown all extensions.
// 4. Shutdown telemetry.
func (srv *Service) Shutdown(ctx context.Context) error {
	// Accumulate errors and proceed with shutting down remaining components.
	var errs error

	// Begin shutdown sequence.
	srv.telemetrySettings.Logger.Info("Starting shutdown...")

	if err := srv.host.ServiceExtensions.NotifyPipelineNotReady(); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to notify that pipeline is not ready: %w", err))
	}

	if err := srv.host.Pipelines.ShutdownAll(ctx, srv.host.Reporter); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown pipelines: %w", err))
	}

	if err := srv.host.ServiceExtensions.Shutdown(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown extensions: %w", err))
	}

	srv.telemetrySettings.Logger.Info("Shutdown complete.")

	errs = multierr.Append(errs, srv.shutdownTelemetry(ctx))

	return errs
}

// Creates extensions.
func (srv *Service) initExtensions(ctx context.Context, cfg extensions.Config) error {
	var err error
	extensionsSettings := extensions.Settings{
		Telemetry:  srv.telemetrySettings,
		BuildInfo:  srv.buildInfo,
		Extensions: srv.host.Extensions,
		ModuleInfo: srv.host.ModuleInfo,
	}
	if srv.host.ServiceExtensions, err = extensions.New(ctx, extensionsSettings, cfg, extensions.WithReporter(srv.host.Reporter)); err != nil {
		return fmt.Errorf("failed to build extensions: %w", err)
	}
	return nil
}

// Creates the pipeline graph.
func (srv *Service) initGraph(ctx context.Context, cfg Config) error {
	// nolint
	if len(cfg.PipelinesWithPipelineID) > 0 {
		cfg.Pipelines = make(pipelines.Config, len(cfg.PipelinesWithPipelineID))
		for k, v := range cfg.PipelinesWithPipelineID {
			cfg.Pipelines[k] = v
		}
	}

	var err error
	if srv.host.Pipelines, err = graph.Build(ctx, graph.Settings{
		Telemetry:        srv.telemetrySettings,
		BuildInfo:        srv.buildInfo,
		ReceiverBuilder:  srv.host.Receivers,
		ProcessorBuilder: srv.host.Processors,
		ExporterBuilder:  srv.host.Exporters,
		ConnectorBuilder: srv.host.Connectors,
		PipelineConfigs:  cfg.Pipelines,
		ReportStatus:     srv.host.Reporter.ReportStatus,
	}); err != nil {
		return fmt.Errorf("failed to build pipelines: %w", err)
	}
	return nil
}

// Logger returns the logger created for this service.
// This is a temporary API that may be removed soon after investigating how the collector should record different events.
func (srv *Service) Logger() *zap.Logger {
	return srv.telemetrySettings.Logger
}

func pdataFromSdk(res *sdkresource.Resource) pcommon.Resource {
	// pcommon.NewResource is the best way to generate a new resource currently and is safe to use outside of tests.
	// Because the resource is signal agnostic, and we need a net new resource, not an existing one, this is the only
	// method of creating it without exposing internal packages.
	pcommonRes := pcommon.NewResource()
	for _, keyValue := range res.Attributes() {
		pcommonRes.Attributes().PutStr(string(keyValue.Key), keyValue.Value.AsString())
	}
	return pcommonRes
}
