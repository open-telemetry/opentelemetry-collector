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
	noopmetric "go.opentelemetry.io/otel/metric/noop"
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
	"go.opentelemetry.io/collector/service/internal/status"
	"go.opentelemetry.io/collector/service/telemetry"
)

// This feature gate is deprecated and will be removed in 1.40.0. Views can now be configured.
var _ = featuregate.GlobalRegistry().MustRegister(
	"telemetry.disableHighCardinalityMetrics",
	featuregate.StageDeprecated,
	featuregate.WithRegisterToVersion("0.133.0"),
	featuregate.WithRegisterDescription(
		"Controls whether the collector should enable potentially high "+
			"cardinality metrics. Deprecated, configure service::telemetry::metrics::views instead."))

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
	//
	// These options will be appended to any options passed to BuildZapLogger.
	//
	// Deprecated [v0.142.0]: use BuildZapLogger instead. This field will be
	// removed in the future, and options must be injected through BuildZapLogger.
	LoggingOptions []zap.Option

	// BuildZapLogger holds an optional function for creating a Zap logger from
	// a zap.Config and options. If this is unspecified, zap.Config.Build will
	// be used.
	//
	// NOTE: in the future this field will be required.
	BuildZapLogger func(zap.Config, ...zap.Option) (*zap.Logger, error)

	// TelemetryFactory is the factory for creating internal telemetry providers.
	TelemetryFactory telemetry.Factory
}

// Service represents the implementation of a component.Host.
type Service struct {
	buildInfo          component.BuildInfo
	telemetrySettings  component.TelemetrySettings
	host               *graph.Host
	collectorConf      *confmap.Conf
	loggerShutdownFunc component.ShutdownFunc
	meterProvider      telemetry.MeterProvider
	tracerProvider     telemetry.TracerProvider
}

// New creates a new Service, its telemetry, and Components.
func New(ctx context.Context, set Settings, cfg Config) (_ *Service, resultErr error) {
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

	if set.TelemetryFactory == nil {
		return nil, errors.New("telemetry factory not provided")
	}

	// Create the resource first. This ensures all telemetry providers
	// (logger, meter, tracer) use the same resource with a consistent service.instance.id.
	telemetrySettings := telemetry.Settings{BuildInfo: set.BuildInfo}
	resource, err := set.TelemetryFactory.CreateResource(ctx, telemetrySettings, cfg.Telemetry)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	telemetrySettings.Resource = &resource

	// Create a function for telemetry providers to build the Zap logger.
	// This injects any LoggingOptions specified in the Settings.
	buildZapLogger := set.BuildZapLogger
	if buildZapLogger == nil {
		buildZapLogger = zap.Config.Build
	}
	if len(set.LoggingOptions) > 0 {
		origBuildZapLogger := buildZapLogger
		buildZapLogger = func(cfg zap.Config, opts ...zap.Option) (*zap.Logger, error) {
			opts = append(opts, set.LoggingOptions...)
			return origBuildZapLogger(cfg, opts...)
		}
	}

	loggerSettings := telemetry.LoggerSettings{
		Settings:       telemetrySettings,
		BuildZapLogger: buildZapLogger,
	}
	logger, loggerShutdownFunc, err := set.TelemetryFactory.CreateLogger(ctx, loggerSettings, cfg.Telemetry)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}
	defer func() {
		if resultErr != nil {
			logger.Error("error found during service initialization", zap.Error(resultErr))
			resultErr = multierr.Append(resultErr, loggerShutdownFunc.Shutdown(ctx))
		}
	}()
	srv.loggerShutdownFunc = loggerShutdownFunc

	meterSettings := telemetry.MeterSettings{
		Settings:     telemetrySettings,
		Logger:       logger,
		DefaultViews: configureViews,
	}
	meterProvider, err := set.TelemetryFactory.CreateMeterProvider(ctx, meterSettings, cfg.Telemetry)
	if err != nil {
		return nil, fmt.Errorf("failed to create meter provider: %w", err)
	}
	defer func() {
		if resultErr != nil {
			resultErr = multierr.Append(resultErr, meterProvider.Shutdown(ctx))
		}
	}()
	srv.meterProvider = meterProvider

	tracerSettings := telemetry.TracerSettings{
		Settings: telemetrySettings,
		Logger:   logger,
	}
	tracerProvider, err := set.TelemetryFactory.CreateTracerProvider(ctx, tracerSettings, cfg.Telemetry)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer provider: %w", err)
	}
	defer func() {
		if resultErr != nil {
			resultErr = multierr.Append(resultErr, tracerProvider.Shutdown(ctx))
		}
	}()
	srv.tracerProvider = tracerProvider

	srv.telemetrySettings = component.TelemetrySettings{
		Logger:         logger,
		MeterProvider:  meterProvider,
		TracerProvider: tracerProvider,
		Resource:       resource,
	}
	srv.host.Reporter = status.NewReporter(srv.host.NotifyComponentStatusChange, func(err error) {
		if errors.Is(err, status.ErrStatusNotReady) {
			logger.Warn("Invalid transition", zap.Error(err))
		}
		// ignore other errors as they represent invalid state transitions and are considered benign.
	})

	err = srv.initGraph(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// process the configuration and initialize the pipeline
	err = srv.initExtensions(ctx, cfg.Extensions)
	if err != nil {
		return nil, err
	}

	if err := proctelemetry.RegisterProcessMetrics(srv.telemetrySettings); err != nil {
		return nil, fmt.Errorf("failed to register process metrics: %w", err)
	}
	return srv, nil
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

	// Shut down telemetry providers in the reverse order of creation,
	// since the tracer and meter providers may use the logger.
	if err := srv.tracerProvider.Shutdown(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown tracer provider: %w", err))
	}
	if err := srv.meterProvider.Shutdown(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown meter provider: %w", err))
	}
	if err := srv.loggerShutdownFunc.Shutdown(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown logger: %w", err))
	}

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
			// Drop duration metric if the level is not detailed
			dropViewOption(&config.ViewSelector{
				MeterName:      ptr("go.opentelemetry.io/collector/processor/processorhelper"),
				InstrumentName: ptr("otelcol_processor_internal_duration"),
			}),
		)
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

	// Batch exporter metrics
	if level < configtelemetry.LevelDetailed {
		scope := ptr("go.opentelemetry.io/collector/exporter/exporterhelper")
		views = append(views, dropViewOption(&config.ViewSelector{
			MeterName:      scope,
			InstrumentName: ptr("otelcol_exporter_queue_batch_send_size_bytes"),
		}))
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

	// Internal graph metrics
	graphScope := ptr("go.opentelemetry.io/collector/service")
	if level < configtelemetry.LevelDetailed {
		views = append(views,
			dropViewOption(&config.ViewSelector{
				MeterName:      graphScope,
				InstrumentName: ptr("otelcol.*.consumed.size"),
			}),
			dropViewOption(&config.ViewSelector{
				MeterName:      graphScope,
				InstrumentName: ptr("otelcol.*.produced.size"),
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
