// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		success bool
	}{
		{
			name: "basic metric telemetry",
			cfg: &Config{
				Metrics: MetricsConfig{
					Level:   configtelemetry.LevelBasic,
					Address: "127.0.0.1:3333",
				},
			},
			success: true,
		},
		{
			name: "invalid metric telemetry",
			cfg: &Config{
				Metrics: MetricsConfig{
					Level:   configtelemetry.LevelBasic,
					Address: "",
				},
			},
			success: false,
		},
		{
			name: "valid metric telemetry with metric readers",
			cfg: &Config{
				Metrics: MetricsConfig{
					Level: configtelemetry.LevelBasic,
					Readers: []MetricReader{
						{Pull: &PullMetricReader{}},
					},
				},
			},
			success: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.success {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// Force the state of feature gate for a test
func setFeatureGateForTest(t testing.TB, gate *featuregate.Gate, enabled bool) func() {
	originalValue := gate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(gate.ID(), enabled))
	return func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(gate.ID(), originalValue))
	}
}

func TestUnmarshalMetricReaderWithGateOff(t *testing.T) {
	defer setFeatureGateForTest(t, obsreportconfig.UseOtelWithSDKConfigurationForInternalTelemetryFeatureGate, false)()
	reader := MetricReader{}
	assert.NoError(t, reader.Unmarshal(confmap.NewFromStringMap(map[string]any{"invalid": "invalid"})))
}

func TestUnmarshalMetricReader(t *testing.T) {
	defer setFeatureGateForTest(t, obsreportconfig.UseOtelWithSDKConfigurationForInternalTelemetryFeatureGate, true)()
	tests := []struct {
		name string
		cfg  *confmap.Conf
		err  string
	}{
		{
			name: "invalid config",
			cfg:  confmap.NewFromStringMap(map[string]any{"invalid": "invalid"}),
			err:  "unsupported metric reader type [invalid]",
		},
		{
			name: "nil config, nothing to do",
		},
		{
			name: "invalid pull reader type with valid prometheus exporter",
			cfg: confmap.NewFromStringMap(map[string]any{"pull/prometheus1": PullMetricReader{
				Exporter: MetricExporter{
					Prometheus: &Prometheus{},
				},
			}}),
			err: "unsupported metric reader type [pull/prometheus1]",
		},
		{
			name: "valid reader type, invalid config",
			cfg:  confmap.NewFromStringMap(map[string]any{"pull": "garbage"}),
			err:  "invalid metric reader configuration",
		},
		{
			name: "valid pull reader type, no exporter",
			cfg:  confmap.NewFromStringMap(map[string]any{"pull": PullMetricReader{}}),
			err:  "invalid exporter configuration",
		},
		{
			name: "valid pull reader type, invalid exporter",
			cfg: confmap.NewFromStringMap(map[string]any{"pull": PullMetricReader{
				Exporter: MetricExporter{
					Prometheus: nil,
				},
			}}),
			err: "invalid exporter configuration",
		},
		{
			name: "valid pull reader type, valid prometheus exporter",
			cfg: confmap.NewFromStringMap(map[string]any{"pull": PullMetricReader{
				Exporter: MetricExporter{
					Prometheus: &Prometheus{},
				},
			}}),
		},
		{
			name: "valid periodic reader type, valid console exporter",
			cfg: confmap.NewFromStringMap(map[string]any{"periodic": PeriodicMetricReader{
				Exporter: MetricExporter{
					Console: Console{},
				},
			}}),
		},
		{
			name: "valid periodic reader type, invalid console exporter",
			cfg: confmap.NewFromStringMap(map[string]any{"periodic": PeriodicMetricReader{
				Exporter: MetricExporter{
					Prometheus: &Prometheus{},
				},
			}}),
			err: "invalid exporter configuration",
		},
		{
			name: "valid periodic reader type, valid otlp exporter",
			cfg: confmap.NewFromStringMap(map[string]any{"periodic": PeriodicMetricReader{
				Exporter: MetricExporter{
					Otlp: &OtlpMetric{},
				},
			}}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := MetricReader{}
			err := reader.Unmarshal(tt.cfg)
			if len(tt.err) > 0 {
				assert.ErrorContains(t, err, tt.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnmarshalSpanProcessorWithGateOff(t *testing.T) {
	defer setFeatureGateForTest(t, obsreportconfig.UseOtelWithSDKConfigurationForInternalTelemetryFeatureGate, false)()
	sp := SpanProcessor{}
	assert.NoError(t, sp.Unmarshal(confmap.NewFromStringMap(map[string]any{"invalid": "invalid"})))
}

func TestUnmarshalSpanProcessor(t *testing.T) {
	defer setFeatureGateForTest(t, obsreportconfig.UseOtelWithSDKConfigurationForInternalTelemetryFeatureGate, true)()
	tests := []struct {
		name string
		cfg  *confmap.Conf
		err  string
	}{
		{
			name: "invalid config",
			cfg:  confmap.NewFromStringMap(map[string]any{"invalid": "invalid"}),
			err:  "unsupported span processor type [invalid]",
		},
		{
			name: "nil config, nothing to do",
		},
		{
			name: "invalid batch processor type with valid console exporter",
			cfg: confmap.NewFromStringMap(map[string]any{"thing": BatchSpanProcessor{
				Exporter: SpanExporter{
					Console: Console{},
				},
			}}),
			err: "unsupported span processor type [thing]",
		},
		{
			name: "valid batch processor, invalid config",
			cfg:  confmap.NewFromStringMap(map[string]any{"batch": "garbage"}),
			err:  "invalid span processor configuration",
		},
		{
			name: "valid batch processor, no exporter",
			cfg:  confmap.NewFromStringMap(map[string]any{"batch": BatchSpanProcessor{}}),
			err:  "invalid exporter configuration",
		},
		{
			name: "valid batch processor, valid console exporter",
			cfg: confmap.NewFromStringMap(map[string]any{"batch": BatchSpanProcessor{
				Exporter: SpanExporter{
					Console: Console{},
				},
			}}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := SpanProcessor{}
			err := processor.Unmarshal(tt.cfg)
			if len(tt.err) > 0 {
				assert.ErrorContains(t, err, tt.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
