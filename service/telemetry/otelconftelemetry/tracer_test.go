// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/service/internal/resource"
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
			name:               "tracer provider",
			wantTracerProvider: &sdktrace.TracerProvider{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buildInfo := component.BuildInfo{}
			sdk, err := NewSDK(context.Background(), &tt.cfg, resource.New(buildInfo, nil))
			require.NoError(t, err)
			defer func() {
				require.NoError(t, sdk.Shutdown(context.Background()))
			}()
			provider, err := newTracerProvider(tt.cfg, sdk)
			require.NoError(t, err)
			require.IsType(t, tt.wantTracerProvider, provider)
		})
	}
}
