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
)

const (
	// supported trace propagators
	traceContextPropagator = "tracecontext"
	b3Propagator           = "b3"
)

var (
	errUnsupportedPropagator = errors.New("unsupported trace propagator")
)

func attributes(set Settings, cfg Config) map[string]interface{} {
	attrs := map[string]interface{}{
		string(semconv.ServiceNameKey):    set.BuildInfo.Command,
		string(semconv.ServiceVersionKey): set.BuildInfo.Version,
	}
	for k, v := range cfg.Resource {
		if v != nil {
			attrs[k] = *v
		}

		// the new value is nil, delete the existing key
		if _, ok := attrs[k]; ok && v == nil {
			delete(attrs, k)
		}
	}
	return attrs
}

// New creates a new Telemetry from Config.
func newTracerProvider(ctx context.Context, set Settings, cfg Config) (trace.TracerProvider, error) {
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
