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
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/internal/localhostgate"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/internal/graph"
	"go.opentelemetry.io/collector/service/internal/proctelemetry"
	"go.opentelemetry.io/collector/service/internal/resource"
	"go.opentelemetry.io/collector/service/internal/servicetelemetry"
	"go.opentelemetry.io/collector/service/internal/status"
	"go.opentelemetry.io/collector/service/telemetry"
)

// Settings holds configuration for building a new Service.
type Settings struct {
	// BuildInfo provides collector start information.
	BuildInfo component.BuildInfo

	// CollectorConf contains the Collector's current configuration
	CollectorConf *confmap.Conf

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
}

// Service represents the implementation of a component.Host.
type Service struct {
	buildInfo         component.BuildInfo
	telemetrySettings servicetelemetry.TelemetrySettings
	host              *serviceHost
	collectorConf     *confmap.Conf
}

// New creates a new Service, its telemetry, and Components.
func New(ctx context.Context, set Settings, cfg Config) (*Service, error) {
	disableHighCard := obsreportconfig.DisableHighCardinalityMetricsfeatureGate.IsEnabled()
	extendedConfig := obsreportconfig.UseOtelWithSDKConfigurationForInternalTelemetryFeatureGate.IsEnabled()
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
	mp, err := newMeterProvider(
		meterProviderSettings{
			res:               res,
			cfg:               cfg.Telemetry.Metrics,
			asyncErrorChannel: set.AsyncErrorChannel,
		},
		disableHighCard,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric provider: %w", err)
	}

	logsAboutMeterProvider(logger, cfg.Telemetry.Metrics, mp, extendedConfig)
	srv.telemetrySettings = servicetelemetry.TelemetrySettings{
		Logger:         logger,
		MeterProvider:  mp,
		TracerProvider: tracerProvider,
		MetricsLevel:   cfg.Telemetry.Metrics.Level,
		// Construct telemetry attributes from build info and config's resource attributes.
		Resource: pcommonRes,
		Status: status.NewReporter(srv.host.notifyComponentStatusChange, func(err error) {
			if errors.Is(err, status.ErrStatusNotReady) {
				logger.Warn("Invalid transition", zap.Error(err))
			}
			// ignore other errors as they represent invalid state transitions and are considered benign.
		}),
	}

	// process the configuration and initialize the pipeline
	if err = srv.initExtensionsAndPipeline(ctx, set, cfg); err != nil {
		// If pipeline initialization fails then shut down telemetry
		err = multierr.Append(err, srv.shutdownTelemetry(ctx))
		return nil, err
	}

	return srv, nil
}

func logsAboutMeterProvider(logger *zap.Logger, cfg telemetry.MetricsConfig, mp metric.MeterProvider, extendedConfig bool) {
	if cfg.Level == configtelemetry.LevelNone || (cfg.Address == "" && len(cfg.Readers) == 0) {
		logger.Info(
			"Skipped telemetry setup.",
			zap.String(zapKeyTelemetryAddress, cfg.Address),
			zap.Stringer(zapKeyTelemetryLevel, cfg.Level),
		)
		return
	}

	if len(cfg.Address) != 0 && extendedConfig {
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
	srv.telemetrySettings.Status.Ready()

	if err := srv.host.serviceExtensions.Start(ctx, srv.host); err != nil {
		return fmt.Errorf("failed to start extensions: %w", err)
	}

	if srv.collectorConf != nil {
		if err := srv.host.serviceExtensions.NotifyConfig(ctx, srv.collectorConf); err != nil {
			return err
		}
	}

	if err := srv.host.pipelines.StartAll(ctx, srv.host); err != nil {
		return fmt.Errorf("cannot start pipelines: %w", err)
	}

	if err := srv.host.serviceExtensions.NotifyPipelineReady(); err != nil {
		return err
	}

	srv.telemetrySettings.Logger.Info("Everything is ready. Begin running and processing data.")
	localhostgate.LogAboutUseLocalHostAsDefault(srv.telemetrySettings.Logger)
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

	errs = multierr.Append(errs, srv.shutdownTelemetry(ctx))

	return errs
}

// Creates extensions and then builds the pipeline graph.
func (srv *Service) initExtensionsAndPipeline(ctx context.Context, set Settings, cfg Config) error {
	var err error
	extensionsSettings := extensions.Settings{
		Telemetry:  srv.telemetrySettings,
		BuildInfo:  srv.buildInfo,
		Extensions: srv.host.extensions,
	}
	if srv.host.serviceExtensions, err = extensions.New(ctx, extensionsSettings, cfg.Extensions); err != nil {
		return fmt.Errorf("failed to build extensions: %w", err)
	}

	pSet := graph.Settings{
		Telemetry:        srv.telemetrySettings,
		BuildInfo:        srv.buildInfo,
		ReceiverBuilder:  set.Receivers,
		ProcessorBuilder: set.Processors,
		ExporterBuilder:  set.Exporters,
		ConnectorBuilder: set.Connectors,
		PipelineConfigs:  cfg.Pipelines,
	}

	if srv.host.pipelines, err = graph.Build(ctx, pSet); err != nil {
		return fmt.Errorf("failed to build pipelines: %w", err)
	}

	if cfg.Telemetry.Metrics.Level != configtelemetry.LevelNone && cfg.Telemetry.Metrics.Address != "" {
		// The process telemetry initialization requires the ballast size, which is available after the extensions are initialized.
		if err = proctelemetry.RegisterProcessMetrics(srv.telemetrySettings, getBallastSize(srv.host)); err != nil {
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
