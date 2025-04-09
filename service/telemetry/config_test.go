// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/featuregate"
)

func TestComponentConfigStruct(t *testing.T) {
	require.NoError(t, componenttest.CheckConfigStruct(NewFactory().CreateDefaultConfig()))
}

func TestUnmarshalDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NoError(t, confmap.New().Unmarshal(&cfg))
	assert.Equal(t, factory.CreateDefaultConfig(), cfg)
}

func TestUnmarshalEmptyMetricReaders(t *testing.T) {
	prev := disableAddressFieldForInternalTelemetryFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(disableAddressFieldForInternalTelemetryFeatureGate.ID(), false))
	defer func() {
		// Restore previous value.
		require.NoError(t, featuregate.GlobalRegistry().Set(disableAddressFieldForInternalTelemetryFeatureGate.ID(), prev))
	}()
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config_empty_readers.yaml"))
	require.NoError(t, err)
	cfg := NewFactory().CreateDefaultConfig()
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Empty(t, cfg.(*Config).Metrics.Readers)
}

func TestUnmarshalConfigDeprecatedAddress(t *testing.T) {
	prev := disableAddressFieldForInternalTelemetryFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(disableAddressFieldForInternalTelemetryFeatureGate.ID(), false))
	defer func() {
		// Restore previous value.
		require.NoError(t, featuregate.GlobalRegistry().Set(disableAddressFieldForInternalTelemetryFeatureGate.ID(), prev))
	}()
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config_deprecated_address.yaml"))
	require.NoError(t, err)
	cfg := NewFactory().CreateDefaultConfig()
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.(*Config).Metrics.Readers, 1)
	assert.Equal(t, "localhost", *cfg.(*Config).Metrics.Readers[0].Pull.Exporter.Prometheus.Host)
	assert.Equal(t, 6666, *cfg.(*Config).Metrics.Readers[0].Pull.Exporter.Prometheus.Port)
}

func TestUnmarshalConfigDeprecatedAddressGateEnabled(t *testing.T) {
	prev := disableAddressFieldForInternalTelemetryFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(disableAddressFieldForInternalTelemetryFeatureGate.ID(), true))
	defer func() {
		// Restore previous value.
		require.NoError(t, featuregate.GlobalRegistry().Set(disableAddressFieldForInternalTelemetryFeatureGate.ID(), prev))
	}()
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config_deprecated_address.yaml"))
	require.NoError(t, err)
	cfg := NewFactory().CreateDefaultConfig()
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.(*Config).Metrics.Readers, 1)
	assert.Equal(t, "localhost", *cfg.(*Config).Metrics.Readers[0].Pull.Exporter.Prometheus.Host)
	assert.Equal(t, 8888, *cfg.(*Config).Metrics.Readers[0].Pull.Exporter.Prometheus.Port)
}

func TestUnmarshalConfigInvalidDeprecatedAddress(t *testing.T) {
	prev := disableAddressFieldForInternalTelemetryFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(disableAddressFieldForInternalTelemetryFeatureGate.ID(), false))
	defer func() {
		// Restore previous value.
		require.NoError(t, featuregate.GlobalRegistry().Set(disableAddressFieldForInternalTelemetryFeatureGate.ID(), prev))
	}()
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config_invalid_deprecated_address.yaml"))
	require.NoError(t, err)
	cfg := NewFactory().CreateDefaultConfig()
	require.Error(t, cm.Unmarshal(&cfg))
}

func TestUnmarshalConfigDeprecatedAddressAndReaders(t *testing.T) {
	prev := disableAddressFieldForInternalTelemetryFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(disableAddressFieldForInternalTelemetryFeatureGate.ID(), false))
	defer func() {
		// Restore previous value.
		require.NoError(t, featuregate.GlobalRegistry().Set(disableAddressFieldForInternalTelemetryFeatureGate.ID(), prev))
	}()
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config_deprecated_address_and_readers.yaml"))
	require.NoError(t, err)
	cfg := NewFactory().CreateDefaultConfig()
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.(*Config).Metrics.Readers, 2)
	assert.Equal(t, "localhost", *cfg.(*Config).Metrics.Readers[0].Pull.Exporter.Prometheus.Host)
	assert.Equal(t, 9999, *cfg.(*Config).Metrics.Readers[0].Pull.Exporter.Prometheus.Port)
	assert.Equal(t, "192.168.0.1", *cfg.(*Config).Metrics.Readers[1].Pull.Exporter.Prometheus.Host)
	assert.Equal(t, 6666, *cfg.(*Config).Metrics.Readers[1].Pull.Exporter.Prometheus.Port)
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		success bool
	}{
		{
			name: "basic metric telemetry",
			cfg: &Config{
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
			name: "invalid metric telemetry",
			cfg: &Config{
				Metrics: MetricsConfig{
					Level: configtelemetry.LevelBasic,
				},
			},
			success: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.success {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
