// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

package service // import "go.opentelemetry.io/collector/service"

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv118 "go.opentelemetry.io/otel/semconv/v1.18.0"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
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
	"go.opentelemetry.io/collector/service/internal/moduleinfo"
	"go.opentelemetry.io/collector/service/internal/proctelemetry"
	"go.opentelemetry.io/collector/service/internal/resource"
	"go.opentelemetry.io/collector/service/internal/status"
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

// disableHighCardinalityMetricsFeatureGate is the feature gate that controls whether the collector should enable
// potentially high cardinality metrics. The gate will be removed when the collector allows for view configuration.
var disableHighCardinalityMetricsFeatureGate = featuregate.GlobalRegistry().MustRegister(
	"telemetry.disableHighCardinalityMetrics",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("controls whether the collector should enable potentially high"+
		"cardinality metrics. The gate will be removed when the collector allows for view configuration."))

// ModuleInfo describes the Go module for a particular component.
type ModuleInfo = moduleinfo.ModuleInfo

// ModuleInfos describes the go module for all components.
type ModuleInfos = moduleinfo.ModuleInfos

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
	ModuleInfos ModuleInfos

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
	loggerProvider    log.LoggerProvider
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

			ModuleInfos:       set.ModuleInfos,
			BuildInfo:         set.BuildInfo,
			AsyncErrorChannel: set.AsyncErrorChannel,
		},
		collectorConf: set.CollectorConf,
	}

	// Fetch data for internal telemetry like instance id and sdk version to provide for internal telemetry.
	res := resource.New(set.BuildInfo, cfg.Telemetry.Resource)
	pcommonRes := pdataFromSdk(res)

	sch := semconv.SchemaURL

	mpConfig := &cfg.Telemetry.Metrics.MeterProvider

	if mpConfig.Views != nil {
		if disableHighCardinalityMetricsFeatureGate.IsEnabled() {
			return nil, errors.New("telemetry.disableHighCardinalityMetrics gate is incompatible with service::telemetry::metrics::views")
		}
	} else {
		mpConfig.Views = configureViews(cfg.Telemetry.Metrics.Level)
	}

	if cfg.Telemetry.Metrics.Level == configtelemetry.LevelNone {
		mpConfig.Readers = []config.MetricReader{}
	}

	sdk, err := config.NewSDK(
		config.WithContext(ctx),
		config.WithOpenTelemetryConfiguration(
			config.OpenTelemetryConfiguration{
				LoggerProvider: &config.LoggerProvider{
					Processors: cfg.Telemetry.Logs.Processors,
				},
				MeterProvider: mpConfig,
				TracerProvider: &config.TracerProvider{
					Processors: cfg.Telemetry.Traces.Processors,
				},
				Resource: &config.Resource{
					SchemaUrl:  &sch,
					Attributes: attributes(res, cfg.Telemetry),
				},
			},
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create SDK: %w", err)
	}

	telFactory := telemetry.NewFactory()
	telset := telemetry.Settings{
		AsyncErrorChannel: set.AsyncErrorChannel,
		BuildInfo:         set.BuildInfo,
		ZapOptions:        set.LoggingOptions,
		SDK:               &sdk,
	}

	logger, lp, err := telFactory.CreateLogger(ctx, telset, &cfg.Telemetry)
	if err != nil {
		err = multierr.Append(err, sdk.Shutdown(ctx))
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}
	srv.loggerProvider = lp

	tracerProvider, err := telFactory.CreateTracerProvider(ctx, telset, &cfg.Telemetry)
	if err != nil {
		err = multierr.Append(err, sdk.Shutdown(ctx))
		return nil, fmt.Errorf("failed to create tracer provider: %w", err)
	}

	logger.Info("Setting up own telemetry...")

	mp, err := telFactory.CreateMeterProvider(ctx, telset, &cfg.Telemetry)
	if err != nil {
		err = multierr.Append(err, sdk.Shutdown(ctx))
		return nil, fmt.Errorf("failed to create meter provider: %w", err)
	}
	srv.telemetrySettings = component.TelemetrySettings{
		Logger:         logger,
		MeterProvider:  mp,
		TracerProvider: tracerProvider,
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

	if cfg.Telemetry.Metrics.Level != configtelemetry.LevelNone && (len(mpConfig.Readers) != 0 || cfg.Telemetry.Metrics.Address != "") {
		if err = proctelemetry.RegisterProcessMetrics(srv.telemetrySettings); err != nil {
			return nil, fmt.Errorf("failed to register process metrics: %w", err)
		}
	}

	logsAboutMeterProvider(logger, cfg.Telemetry.Metrics, mp)

	return srv, nil
}

func logsAboutMeterProvider(logger *zap.Logger, cfg telemetry.MetricsConfig, mp metric.MeterProvider) {
	if cfg.Level == configtelemetry.LevelNone || len(cfg.Readers) == 0 {
		logger.Info("Skipped telemetry setup.")
		return
	}

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

	if prov, ok := srv.loggerProvider.(shutdownable); ok {
		if shutdownErr := prov.Shutdown(ctx); shutdownErr != nil {
			err = multierr.Append(err, fmt.Errorf("failed to shutdown logger provider: %w", shutdownErr))
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
	}
	if srv.host.ServiceExtensions, err = extensions.New(ctx, extensionsSettings, cfg, extensions.WithReporter(srv.host.Reporter)); err != nil {
		return fmt.Errorf("failed to build extensions: %w", err)
	}
	return nil
}

// Creates the pipeline graph.
func (srv *Service) initGraph(ctx context.Context, cfg Config) error {
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

func dropViewOption(selector *config.ViewSelector) config.View {
	return config.View{
		Selector: selector,
		Stream: &config.ViewStream{
			Aggregation: &config.ViewStreamAggregation{
				Drop: config.ViewStreamAggregationDrop{},
			},
		},
	}
}

func configureViews(level configtelemetry.Level) []config.View {
	views := []config.View{}

	if level < configtelemetry.LevelDetailed {
		// Drop all otelhttp and otelgrpc metrics if the level is not detailed.
		views = append(views,
			dropViewOption(&config.ViewSelector{
				MeterName: ptr("go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"),
			}),
			dropViewOption(&config.ViewSelector{
				MeterName: ptr("go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"),
			}),
		)
	}

	// Make sure to add the AttributeKeys view after the AggregationDrop view:
	// Only the first view outputting a given metric identity is actually used, so placing the
	// AttributeKeys view first would never drop the metrics regadless of level.
	if disableHighCardinalityMetricsFeatureGate.IsEnabled() {
		views = append(views, []config.View{
			{
				Selector: &config.ViewSelector{
					MeterName: ptr("go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"),
				},
				Stream: &config.ViewStream{
					AttributeKeys: &config.IncludeExclude{
						Excluded: []string{
							string(semconv118.NetSockPeerAddrKey),
							string(semconv118.NetSockPeerPortKey),
							string(semconv118.NetSockPeerNameKey),
						},
					},
				},
			},
			{
				Selector: &config.ViewSelector{
					MeterName: ptr("go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"),
				},
				Stream: &config.ViewStream{
					AttributeKeys: &config.IncludeExclude{
						Excluded: []string{
							string(semconv118.NetHostNameKey),
							string(semconv118.NetHostPortKey),
						},
					},
				},
			},
		}...)
	}

	// otel-arrow library metrics
	// See https://github.com/open-telemetry/otel-arrow/blob/c39257/pkg/otel/arrow_record/consumer.go#L174-L176
	if level < configtelemetry.LevelNormal {
		scope := ptr("otel-arrow/pkg/otel/arrow_record")
		views = append(views,
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("arrow_batch_records"),
			}),
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("arrow_schema_resets"),
			}),
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("arrow_memory_inuse"),
			}),
		)
	}

	// contrib's internal/otelarrow/netstats metrics
	// See
	// - https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/a25f05/internal/otelarrow/netstats/netstats.go#L130
	// - https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/a25f05/internal/otelarrow/netstats/netstats.go#L165
	if level < configtelemetry.LevelDetailed {
		scope := ptr("github.com/open-telemetry/opentelemetry-collector-contrib/internal/otelarrow/netstats")

		views = append(views,
			// Compressed size metrics.
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("otelcol_*_compressed_size"),
			}),
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("otelcol_*_compressed_size"),
			}),

			// makeRecvMetrics for exporters.
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("otelcol_exporter_recv"),
			}),
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("otelcol_exporter_recv_wire"),
			}),

			// makeSentMetrics for receivers.
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("otelcol_receiver_sent"),
			}),
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("otelcol_receiver_sent_wire"),
			}),
		)
	}

	// Batch processor metrics
	scope := ptr("go.opentelemetry.io/collector/processor/batchprocessor")
	if level < configtelemetry.LevelNormal {
		views = append(views, dropViewOption(&config.ViewSelector{
			MeterName: scope,
		}))
	} else if level < configtelemetry.LevelDetailed {
		views = append(views, dropViewOption(&config.ViewSelector{
			MeterName:      scope,
			InstrumentName: ptr("otelcol_processor_batch_batch_send_size_bytes"),
		}))
	}
	return views
}

func ptr[T any](v T) *T {
	return &v
}

// Validate verifies the graph by calling the internal graph.Build.
func Validate(ctx context.Context, set Settings, cfg Config) error {
	tel := component.TelemetrySettings{
		Logger:         zap.NewNop(),
		TracerProvider: nooptrace.NewTracerProvider(),
		MeterProvider:  noopmetric.NewMeterProvider(),
		Resource:       pcommon.NewResource(),
	}
	_, err := graph.Build(ctx, graph.Settings{
		Telemetry:        tel,
		BuildInfo:        set.BuildInfo,
		ReceiverBuilder:  builders.NewReceiver(set.ReceiversConfigs, set.ReceiversFactories),
		ProcessorBuilder: builders.NewProcessor(set.ProcessorsConfigs, set.ProcessorsFactories),
		ExporterBuilder:  builders.NewExporter(set.ExportersConfigs, set.ExportersFactories),
		ConnectorBuilder: builders.NewConnector(set.ConnectorsConfigs, set.ConnectorsFactories),
		PipelineConfigs:  cfg.Pipelines,
	})
	if err != nil {
		return fmt.Errorf("failed to build pipelines: %w", err)
	}
	return nil
}
