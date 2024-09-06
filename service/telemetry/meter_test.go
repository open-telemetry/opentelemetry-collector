// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"
import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/service/telemetry/internal"
)

func TestNewMeterProvider(t *testing.T) {
	tests := []struct {
		name                   string
		wantMeterProvider      any
		disableHighCardinality bool
		cfg                    Config
	}{
		{
			name: "metrics level none",
			cfg: Config{
				Metrics: MetricsConfig{
					Level: configtelemetry.LevelNone,
				},
			},
			wantMeterProvider: noop.MeterProvider{},
		},
		{
			name:                   "noop tracer feature gate",
			cfg:                    Config{},
			disableHighCardinality: true,
			wantMeterProvider:      noop.MeterProvider{},
		},
		{
			name: "meter provider from address",
			cfg: Config{
				Metrics: MetricsConfig{
					Address: ":8888",
				},
			},
			wantMeterProvider: &sdkmetric.MeterProvider{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previousValue := obsreportconfig.DisableHighCardinalityMetricsfeatureGate.IsEnabled()
			require.NoError(t, featuregate.GlobalRegistry().Set(obsreportconfig.DisableHighCardinalityMetricsfeatureGate.ID(), tt.disableHighCardinality))
			defer func() {
				require.NoError(t, featuregate.GlobalRegistry().Set(obsreportconfig.DisableHighCardinalityMetricsfeatureGate.ID(), previousValue))
			}()
			provider, shutdown, err := newMeterProvider(context.TODO(), internal.Settings{}, tt.cfg)
			require.NoError(t, err)
			require.IsType(t, tt.wantMeterProvider, provider)
			defer func() {
				require.NoError(t, shutdown(context.Background()))
			}()
		})
	}
}
