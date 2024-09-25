// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/globalgates"
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
			previousValue := globalgates.NoopTracerProvider.IsEnabled()
			// expect error due to deprecated flag
			require.NoError(t, featuregate.GlobalRegistry().Set(globalgates.NoopTracerProvider.ID(), tt.noopTracerGate))
			defer func() {
				require.NoError(t, featuregate.GlobalRegistry().Set(globalgates.NoopTracerProvider.ID(), previousValue))
			}()
			provider, err := newTracerProvider(context.TODO(), Settings{}, tt.cfg)
			require.NoError(t, err)
			require.IsType(t, tt.wantTracerProvider, provider)
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
