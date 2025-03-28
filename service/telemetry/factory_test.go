// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/featuregate"
)

func TestDefaultConfig(t *testing.T) {
	tests := []struct {
		expected string
		gate     bool
	}{
		{
			expected: "localhost",
			gate:     true,
		},
		{
			expected: "",
			gate:     false,
		},
	}

	for _, tt := range tests {
		t.Run("UseLocalHostAsDefaultMetricsAddress/"+strconv.FormatBool(tt.gate), func(t *testing.T) {
			prev := useLocalHostAsDefaultMetricsAddressFeatureGate.IsEnabled()
			require.NoError(t, featuregate.GlobalRegistry().Set(useLocalHostAsDefaultMetricsAddressFeatureGate.ID(), tt.gate))
			defer func() {
				// Restore previous value.
				require.NoError(t, featuregate.GlobalRegistry().Set(useLocalHostAsDefaultMetricsAddressFeatureGate.ID(), prev))
			}()
			cfg := NewFactory().CreateDefaultConfig()
			require.Len(t, cfg.(*Config).Metrics.Readers, 1)
			assert.Equal(t, tt.expected, *cfg.(*Config).Metrics.Readers[0].Pull.Exporter.Prometheus.Host)
			assert.Equal(t, 8888, *cfg.(*Config).Metrics.Readers[0].Pull.Exporter.Prometheus.Port)
		})
	}
}

func TestTelemetryConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		success bool
	}{
		{
			name: "Valid config",
			cfg: &Config{
				Logs: LogsConfig{
					Level:    zapcore.DebugLevel,
					Encoding: "console",
				},
				Metrics: MetricsConfig{
					Level: configtelemetry.LevelBasic,
					MeterProvider: config.MeterProvider{
						Readers: []config.MetricReader{
							{
								Pull: &config.PullMetricReader{Exporter: config.PullMetricExporter{Prometheus: &config.Prometheus{
									Host: newPtr("127.0.0.1"),
									Port: newPtr(3333),
								}}},
							},
						},
					},
				},
			},
			success: true,
		},
		{
			name: "Invalid config",
			cfg: &Config{
				Logs: LogsConfig{
					Level: zapcore.DebugLevel,
				},
				Metrics: MetricsConfig{
					Level: configtelemetry.LevelBasic,
					MeterProvider: config.MeterProvider{
						Readers: []config.MetricReader{
							{
								Pull: &config.PullMetricReader{Exporter: config.PullMetricExporter{Prometheus: &config.Prometheus{
									Host: newPtr("127.0.0.1"),
									Port: newPtr(3333),
								}}},
							},
						},
					},
				},
			},
			success: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFactory()
			set := Settings{ZapOptions: []zap.Option{}}
			logger, _, err := f.CreateLogger(context.Background(), set, tt.cfg)
			if tt.success {
				require.NoError(t, err)
				assert.NotNil(t, logger)
			} else {
				require.Error(t, err)
				assert.Nil(t, logger)
			}
		})
	}
}

func TestSampledLogger(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
	}{
		{
			name: "Default sampling",
			cfg: &Config{
				Logs: LogsConfig{
					Encoding: "console",
				},
			},
		},
		{
			name: "Custom sampling",
			cfg: &Config{
				Logs: LogsConfig{
					Level:    zapcore.DebugLevel,
					Encoding: "console",
					Sampling: &LogsSamplingConfig{
						Enabled:    true,
						Tick:       1 * time.Second,
						Initial:    100,
						Thereafter: 100,
					},
				},
			},
		},
		{
			name: "Disable sampling",
			cfg: &Config{
				Logs: LogsConfig{
					Level:    zapcore.DebugLevel,
					Encoding: "console",
					Sampling: &LogsSamplingConfig{
						Enabled: false,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFactory()
			ctx := context.Background()
			set := Settings{ZapOptions: []zap.Option{}}
			logger, _, err := f.CreateLogger(ctx, set, tt.cfg)
			require.NoError(t, err)
			assert.NotNil(t, logger)
		})
	}
}
