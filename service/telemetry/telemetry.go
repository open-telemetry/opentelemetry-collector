// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"

	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/service/internal/proctelemetry"

	ocmetric "go.opencensus.io/metric"
	"go.opencensus.io/metric/metricproducer"

	"go.opentelemetry.io/contrib/config"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/internal/resource"
)

const (
	// supported trace propagators
	traceContextPropagator = "tracecontext"
	b3Propagator           = "b3"
	zapKeyTelemetryAddress = "address"
	zapKeyTelemetryLevel   = "level"
)

var (
	errUnsupportedPropagator = errors.New("unsupported trace propagator")
)

type Telemetry struct {
	logger         *zap.Logger
	tracerProvider *sdktrace.TracerProvider

	meterProvider metric.MeterProvider
	servers       []*http.Server
	ocRegistry    *ocmetric.Registry
}

func (t *Telemetry) TracerProvider() trace.TracerProvider {
	return t.tracerProvider
}

func (t *Telemetry) MeterProvider() metric.MeterProvider {
	return t.meterProvider
}

func (t *Telemetry) Logger() *zap.Logger {
	return t.logger
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	metricproducer.GlobalManager().DeleteProducer(t.ocRegistry)

	var errs error
	for _, server := range t.servers {
		if server != nil {
			errs = multierr.Append(errs, server.Close())
		}
	}

	if mp, ok := t.meterProvider.(*sdkmetric.MeterProvider); ok {
		errs = multierr.Append(errs, mp.Shutdown(ctx))
	}

	multierr.Append(errs, t.tracerProvider.Shutdown(ctx))

	// TODO: Sync logger.
	return errs
}

// Settings holds configuration for building Telemetry.
type Settings struct {
	BuildInfo         component.BuildInfo
	AsyncErrorChannel chan error
	ZapOptions        []zap.Option
}

type factory struct {
	disableHighCardinality bool
	extendedConfig         bool
}

func newFactory(disableHighCardinality bool, extendedConfig bool) *factory {
	return &factory{
		disableHighCardinality: disableHighCardinality,
		extendedConfig:         extendedConfig,
	}
}

func New(ctx context.Context, set Settings, cfg Config) (*Telemetry, error) {
	disableHighCardinality := obsreportconfig.DisableHighCardinalityMetricsfeatureGate.IsEnabled()
	extendedConfig := obsreportconfig.UseOtelWithSDKConfigurationForInternalTelemetryFeatureGate.IsEnabled()
	factory := newFactory(disableHighCardinality, extendedConfig)
	return factory.New(ctx, set, cfg)
}

// New creates a new Telemetry from Config.
func (f *factory) New(ctx context.Context, set Settings, cfg Config) (*Telemetry, error) {
	logger, err := newLogger(cfg.Logs, set.ZapOptions)
	if err != nil {
		return nil, err
	}

	tp, err := newTracerProvider(ctx, set, cfg)
	if err != nil {
		return nil, err
	}

	res := resource.New(set.BuildInfo, cfg.Resource)
	mp, ocRegistry, servers, err := f.newMeterProvider(set, cfg.Metrics, res, logger)
	if err != nil {
		return nil, err
	}
	return &Telemetry{
		logger:         logger,
		tracerProvider: tp,
		meterProvider:  mp,
		ocRegistry:     ocRegistry,
		servers:        servers,
	}, nil
}

func newTracerProvider(ctx context.Context, set Settings, cfg Config) (*sdktrace.TracerProvider, error) {
	opts := []sdktrace.TracerProviderOption{sdktrace.WithSampler(alwaysRecord())}
	for _, processor := range cfg.Traces.Processors {
		sp, err := proctelemetry.InitSpanProcessor(ctx, processor)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithSpanProcessor(sp))
	}

	res := resource.New(set.BuildInfo, cfg.Resource)
	tp, err := proctelemetry.InitTracerProvider(res, opts)
	if err != nil {
		return nil, err
	}

	if tp, err := textMapPropagatorFromConfig(cfg.Traces.Propagators); err == nil {
		otel.SetTextMapPropagator(tp)
	} else {
		return nil, err
	}

	return tp, nil
}

func textMapPropagatorFromConfig(props []string) (propagation.TextMapPropagator, error) {
	var textMapPropagators []propagation.TextMapPropagator
	for _, prop := range props {
		switch prop {
		case traceContextPropagator:
			textMapPropagators = append(textMapPropagators, propagation.TraceContext{})
		case b3Propagator:
			textMapPropagators = append(textMapPropagators, b3.New())
		default:
			return nil, errUnsupportedPropagator
		}
	}
	return propagation.NewCompositeTextMapPropagator(textMapPropagators...), nil
}

func newLogger(cfg LogsConfig, options []zap.Option) (*zap.Logger, error) {
	// Copied from NewProductionConfig.
	zapCfg := &zap.Config{
		Level:             zap.NewAtomicLevelAt(cfg.Level),
		Development:       cfg.Development,
		Encoding:          cfg.Encoding,
		EncoderConfig:     zap.NewProductionEncoderConfig(),
		OutputPaths:       cfg.OutputPaths,
		ErrorOutputPaths:  cfg.ErrorOutputPaths,
		DisableCaller:     cfg.DisableCaller,
		DisableStacktrace: cfg.DisableStacktrace,
		InitialFields:     cfg.InitialFields,
	}

	if zapCfg.Encoding == "console" {
		// Human-readable timestamps for console format of logs.
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	logger, err := zapCfg.Build(options...)
	if err != nil {
		return nil, err
	}
	if cfg.Sampling != nil && cfg.Sampling.Enabled {
		logger = newSampledLogger(logger, cfg.Sampling)
	}

	return logger, nil
}

func newSampledLogger(logger *zap.Logger, sc *LogsSamplingConfig) *zap.Logger {
	// Create a logger that samples every Nth message after the first M messages every S seconds
	// where N = sc.Thereafter, M = sc.Initial, S = sc.Tick.
	opts := zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewSamplerWithOptions(
			core,
			sc.Tick,
			sc.Initial,
			sc.Thereafter,
		)
	})
	return logger.WithOptions(opts)
}

func (f *factory) newMeterProvider(set Settings, cfg MetricsConfig, res *sdkresource.Resource, logger *zap.Logger) (metric.MeterProvider, *ocmetric.Registry, []*http.Server, error) {
	if cfg.Level == configtelemetry.LevelNone || (cfg.Address == "" && len(cfg.Readers) == 0) {
		logger.Info(
			"Skipping telemetry setup.",
			zap.String(zapKeyTelemetryAddress, cfg.Address),
			zap.String(zapKeyTelemetryLevel, cfg.Level.String()),
		)
		return noop.NewMeterProvider(), nil, nil, nil
	}

	// Initialize the ocRegistry, still used by the process metrics.
	ocRegistry := ocmetric.NewRegistry()
	servers := []*http.Server{}

	if len(cfg.Address) != 0 {
		if f.extendedConfig {
			logger.Warn("service::telemetry::metrics::address is being deprecated in favor of service::telemetry::metrics::readers")
		}
		host, port, err := net.SplitHostPort(cfg.Address)
		if err != nil {
			return nil, nil, nil, err
		}
		portInt, err := strconv.Atoi(port)
		if err != nil {
			return nil, nil, nil, err
		}
		if cfg.Readers == nil {
			cfg.Readers = []config.MetricReader{}
		}
		cfg.Readers = append(cfg.Readers, config.MetricReader{
			Pull: &config.PullMetricReader{
				Exporter: config.MetricExporter{
					Prometheus: &config.Prometheus{
						Host: &host,
						Port: &portInt,
					},
				},
			},
		})
	}

	metricproducer.GlobalManager().AddProducer(ocRegistry)
	opts := []sdkmetric.Option{}
	for _, reader := range cfg.Readers {
		// https://github.com/open-telemetry/opentelemetry-collector/issues/8045
		r, server, err := proctelemetry.InitMetricReader(context.Background(), reader, set.AsyncErrorChannel)
		if err != nil {
			return nil, nil, nil, err
		}
		if server != nil {
			servers = append(servers, server)
			logger.Info(
				"Serving metrics",
				zap.String(zapKeyTelemetryAddress, server.Addr),
				zap.String(zapKeyTelemetryLevel, cfg.Level.String()),
			)
		}
		opts = append(opts, sdkmetric.WithReader(r))
	}

	mp, err := proctelemetry.InitOpenTelemetry(res, opts, f.disableHighCardinality)
	if err != nil {
		return nil, nil, nil, err
	}

	return mp, ocRegistry, servers, nil
}
