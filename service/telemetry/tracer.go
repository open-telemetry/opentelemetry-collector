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
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"go.opentelemetry.io/otel/trace/noop"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/globalgates"
)

const (
	// supported trace propagators
	traceContextPropagator = "tracecontext"
	b3Propagator           = "b3"
)

var (
	errUnsupportedPropagator = errors.New("unsupported trace propagator")
)

type noopNoContextTracer struct {
	embedded.Tracer
}

var noopSpan = noop.Span{}

func (n *noopNoContextTracer) Start(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
	return ctx, noopSpan
}

type noopNoContextTracerProvider struct {
	embedded.TracerProvider
}

func (n *noopNoContextTracerProvider) Tracer(_ string, _ ...trace.TracerOption) trace.Tracer {
	return &noopNoContextTracer{}
}

// New creates a new Telemetry from Config.
func newTracerProvider(ctx context.Context, set Settings, cfg Config) (trace.TracerProvider, error) {
	if globalgates.NoopTracerProvider.IsEnabled() || cfg.Traces.Level == configtelemetry.LevelNone {
		return &noopNoContextTracerProvider{}, nil
	}

	sch := semconv.SchemaURL
	res := config.Resource{
		SchemaUrl:  &sch,
		Attributes: attributes(set, cfg),
	}

	sdk, err := config.NewSDK(
		config.WithContext(ctx),
		config.WithOpenTelemetryConfiguration(
			config.OpenTelemetryConfiguration{
				Resource: &res,
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

	return sdk.TracerProvider(), nil
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
