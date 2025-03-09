// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"testing"

	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/featuregate"
)

func TestNewTracerProvider(t *testing.T) {
	tests := []struct {
		name               string
		wantTracerProvider any
		noopTracerGate     bool
		cfg                Config
	}{
		{
			name: "trace level none",
			cfg: Config{
				Traces: TracesConfig{
					Level: configtelemetry.LevelNone,
				},
			},
			wantTracerProvider: &noopNoContextTracerProvider{},
		},
		{
			name:               "noop tracer feature gate",
			cfg:                Config{},
			noopTracerGate:     true,
			wantTracerProvider: &noopNoContextTracerProvider{},
		},
		{
			name:               "tracer provider",
			wantTracerProvider: &sdktrace.TracerProvider{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previousValue := noopTracerProvider.IsEnabled()
			// expect error due to deprecated flag
			require.NoError(t, featuregate.GlobalRegistry().Set(noopTracerProvider.ID(), tt.noopTracerGate))
			defer func() {
				require.NoError(t, featuregate.GlobalRegistry().Set(noopTracerProvider.ID(), previousValue))
			}()
			sdk, err := config.NewSDK(config.WithOpenTelemetryConfiguration(config.OpenTelemetryConfiguration{TracerProvider: &config.TracerProvider{
				Processors: tt.cfg.Traces.Processors,
			}}))
			require.NoError(t, err)
			provider, err := newTracerProvider(Settings{SDK: &sdk}, tt.cfg)
			require.NoError(t, err)
			require.IsType(t, tt.wantTracerProvider, provider)
		})
	}
}
