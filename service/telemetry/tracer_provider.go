// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"errors"

	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/collector/service/internal/resource"
)

const (
	// supported trace propagators
	traceContextPropagator = "tracecontext"
	b3Propagator           = "b3"
)

var (
	errUnsupportedPropagator = errors.New("unsupported trace propagator")
)

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

func newTracerProvider(ctx context.Context, set Settings, cfg Config) (*sdktrace.TracerProvider, error) {
	res := resource.New(set.BuildInfo, cfg.Resource)
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		// needed for supporting the zpages extension
		sdktrace.WithSampler(alwaysRecord()),
	}
	for _, processor := range cfg.Traces.Processors {
		sp, err := newSpanProcessor(ctx, processor)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithSpanProcessor(sp))
	}

	if tp, err := textMapPropagatorFromConfig(cfg.Traces.Propagators); err == nil {
		otel.SetTextMapPropagator(tp)
	} else {
		return nil, err
	}

	return sdktrace.NewTracerProvider(opts...), nil
}
