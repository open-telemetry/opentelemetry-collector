package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/config/configtelemetry"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/service/internal/resource"
)

func newProviders(ctx context.Context, set Settings, cfg telemetryConfig) (_ Providers, resultErr error) {
	res := resource.New(set.BuildInfo, cfg.Resource)
	sdk, err := newSDK(ctx, set, cfg, res)
	if err != nil {
		return Providers{}, fmt.Errorf("failed to create SDK: %w", err)
	}
	defer func() {
		if resultErr != nil {
			resultErr = multierr.Append(resultErr, sdk.Shutdown(ctx))
		}
	}()

	logger, loggerProvider, err := newLogger(ctx, set, cfg, res, sdk)
	if err != nil {
		return Providers{}, fmt.Errorf("failed to create logger provider: %w", err)
	}

	var meterProvider metric.MeterProvider
	if cfg.Metrics.Level == configtelemetry.LevelNone || len(cfg.Metrics.Readers) == 0 {
		logger.Info("Internal metrics telemetry disabled")
		meterProvider = noopmetric.NewMeterProvider()
	} else {
		meterProvider = sdk.MeterProvider()
	}

	var tracerProvider trace.TracerProvider
	if noopTracerProvider.IsEnabled() || cfg.Traces.Level == configtelemetry.LevelNone {
		logger.Info("Internal trace telemetry disabled")
		tracerProvider = &noopNoContextTracerProvider{}
	} else {
		propagator, err := textMapPropagatorFromConfig(cfg.Traces.Propagators)
		if err != nil {
			return Providers{}, err
		}
		otel.SetTextMapPropagator(propagator)
		tracerProvider = sdk.TracerProvider()
	}

	return Providers{
		Logger:         logger,
		LoggerProvider: loggerProvider,
		MeterProvider:  meterProvider,
		TracerProvider: tracerProvider,
	}, nil
}

func newSDK(ctx context.Context, set Settings, cfg telemetryConfig, res *sdkresource.Resource) (*config.SDK, error) {
	sch := semconv.SchemaURL

	mpConfig := &cfg.Metrics.MeterProvider
	if mpConfig.Views == nil {
		mpConfig.Views = configureViews(cfg.Metrics.Level)
	}
	if cfg.Metrics.Level == configtelemetry.LevelNone {
		mpConfig.Readers = []config.MetricReader{}
	}

	sdk, err := config.NewSDK(
		config.WithContext(ctx),
		config.WithOpenTelemetryConfiguration(
			config.OpenTelemetryConfiguration{
				LoggerProvider: &config.LoggerProvider{
					Processors: cfg.Logs.Processors,
				},
				MeterProvider:  mpConfig,
				TracerProvider: &cfg.Traces.TracerProvider,
				Resource: &config.Resource{
					SchemaUrl:  &sch,
					Attributes: attributes(res, cfg),
				},
			},
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create SDK: %w", err)
	}
	return &sdk, nil
}
