// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"
	"errors"
	"fmt"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"go.opentelemetry.io/otel/trace/noop"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/service/telemetry"
)

var _ = featuregate.GlobalRegistry().MustRegister("service.noopTracerProvider",
	featuregate.StageDeprecated,
	featuregate.WithRegisterFromVersion("v0.107.0"),
	featuregate.WithRegisterToVersion("v0.109.0"),
	featuregate.WithRegisterDescription("Sets a Noop OpenTelemetry TracerProvider to reduce memory allocations. This featuregate is incompatible with the zPages extension."))

const (
	// supported trace propagators
	traceContextPropagator = "tracecontext"
	b3Propagator           = "b3"
)

func createTracerProvider(
	ctx context.Context,
	set telemetry.TracerSettings,
	componentConfig component.Config,
) (telemetry.TracerProvider, error) {
	cfg := componentConfig.(*Config)
	if cfg.Traces.Level == configtelemetry.LevelNone {
		set.Logger.Info("Internal trace telemetry disabled")
		return &noopNoContextTracerProvider{}, nil
	}

	propagator, err := textMapPropagatorFromConfig(cfg.Traces.Propagators)
	if err != nil {
		return nil, fmt.Errorf("error creating propagator: %w", err)
	}
	otel.SetTextMapPropagator(propagator)

	attrs := pcommonAttrsToOTelAttrs(set.Resource)
	res := sdkresource.NewWithAttributes("", attrs...)
	sdk, err := newSDK(ctx, res, config.OpenTelemetryConfiguration{
		TracerProvider: &cfg.Traces.TracerProvider,
	})
	if err != nil {
		return nil, err
	}
	return sdk.TracerProvider().(telemetry.TracerProvider), nil
}

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

func (n *noopNoContextTracerProvider) Shutdown(_ context.Context) error {
	return nil
}

func (n *noopNoContextTracerProvider) Tracer(_ string, _ ...trace.TracerOption) trace.Tracer {
	return &noopNoContextTracer{}
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
