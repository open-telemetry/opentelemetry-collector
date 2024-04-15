// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"errors"

	"go.opentelemetry.io/contrib/config"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
)

const (
	// supported trace propagators
	traceContextPropagator = "tracecontext"
	b3Propagator           = "b3"
)

var (
	errUnsupportedPropagator = errors.New("unsupported trace propagator")
)

type Telemetry struct {
	logger         *zap.Logger
	tracerProvider trace.TracerProvider
}

func (t *Telemetry) TracerProvider() trace.TracerProvider {
	return t.tracerProvider
}

func (t *Telemetry) Logger() *zap.Logger {
	return t.logger
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	// TODO: Sync logger.
	if tp, ok := t.tracerProvider.(*sdktrace.TracerProvider); ok {
		return multierr.Combine(
			tp.Shutdown(ctx),
		)
	}
	// should this return an error?
	return nil
}

// Settings holds configuration for building Telemetry.
type Settings struct {
	BuildInfo  component.BuildInfo
	ZapOptions []zap.Option
}

// New creates a new Telemetry from Config.
func New(ctx context.Context, set Settings, cfg Config) (*Telemetry, error) {
	logger, err := newLogger(cfg.Logs, set.ZapOptions)
	if err != nil {
		return nil, err
	}

	sdk, err := config.NewSDK(
		config.WithContext(ctx),
		config.WithOpenTelemetryConfiguration(
			config.OpenTelemetryConfiguration{
				TracerProvider: &config.TracerProvider{
					Processors: cfg.Traces.Processors,
					// TODO: once https://github.com/open-telemetry/opentelemetry-configuration/issues/83 is resolved,
					// configuration for sampler should be done here via something like the following:
					//
					// Sampler: &config.Sampler{
					// 	ParentBased: &config.SamplerParentBased{
					// 		LocalParentSampled: &config.Sampler{
					// 			AlwaysOn: config.SamplerAlwaysOn{},
					// 		},
					// 		LocalParentNotSampled: &config.Sampler{
					//	        RecordOnly: config.SamplerRecordOnly{},
					//      },
					// 		RemoteParentSampled: &config.Sampler{
					// 			AlwaysOn: config.SamplerAlwaysOn{},
					// 		},
					// 		RemoteParentNotSampled: &config.Sampler{
					//	        RecordOnly: config.SamplerRecordOnly{},
					//      },
					// 	},
					// },
				},
			},
		),
	)

	if err != nil {
		return nil, err
	}

	if tp, err := textMapPropagatorFromConfig(cfg.Traces.Propagators); err == nil {
		otel.SetTextMapPropagator(tp)
	} else {
		return nil, err
	}

	return &Telemetry{
		logger:         logger,
		tracerProvider: sdk.TracerProvider(),
	}, nil
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
