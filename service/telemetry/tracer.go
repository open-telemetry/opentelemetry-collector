// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"errors"

	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"go.opentelemetry.io/otel/trace/noop"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/featuregate"
)

var noopTracerProvider = featuregate.GlobalRegistry().MustRegister("service.noopTracerProvider",
	featuregate.StageAlpha,
	featuregate.WithRegisterFromVersion("v0.107.0"),
	featuregate.WithRegisterToVersion("v0.109.0"),
	featuregate.WithRegisterDescription("Sets a Noop OpenTelemetry TracerProvider to reduce memory allocations. This featuregate is incompatible with the zPages extension."))

const (
	// supported trace propagators
	traceContextPropagator = "tracecontext"
	b3Propagator           = "b3"
)

var errUnsupportedPropagator = errors.New("unsupported trace propagator")

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

// newTracerProvider creates a new TracerProvider from Config.
func newTracerProvider(set Settings, cfg Config) (trace.TracerProvider, error) {
	if noopTracerProvider.IsEnabled() || cfg.Traces.Level == configtelemetry.LevelNone {
		return &noopNoContextTracerProvider{}, nil
	}

	if tp, err := textMapPropagatorFromConfig(cfg.Traces.Propagators); err == nil {
		otel.SetTextMapPropagator(tp)
	} else {
		return nil, err
	}

	if set.SDK != nil {
		return set.SDK.TracerProvider(), nil
	}
	return nil, errors.New("no sdk set")
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
