// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pipelineprocessor

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/processor/pipelineprocessor/internal/metadata"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	sub, err := cm.Sub(component.NewIDWithName(metadata.Type, "").String())
	require.NoError(t, err)
	require.NoError(t, sub.Unmarshal(cfg))

	expected := &Config{
		TimeoutConfig: exporterhelper.TimeoutConfig{
			Timeout: 10 * time.Second,
		},
		QueueConfig: exporterhelper.QueueConfig{
			Enabled:      true,
			NumConsumers: 10,
			QueueSize:    1000,
			Sizer:        exporterhelper.RequestSizerTypeRequests,
			Batch: &exporterhelper.BatchConfig{
				FlushTimeout: 1 * time.Second,
				MinSize:      100,
				MaxSize:      1000,
			},
		},
		RetryConfig: configretry.BackOffConfig{
			Enabled:             true,
			InitialInterval:     5 * time.Second,
			MaxInterval:         30 * time.Second,
			MaxElapsedTime:      5 * time.Minute,
			Multiplier:          1.5,
			RandomizationFactor: 0.5,
		},
	}
	assert.Equal(t, expected, cfg)
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				TimeoutConfig: exporterhelper.TimeoutConfig{
					Timeout: 5 * time.Second,
				},
				QueueConfig: exporterhelper.NewDefaultQueueConfig(),
				RetryConfig: configretry.NewDefaultBackOffConfig(),
			},
			wantErr: false,
		},
		{
			name: "invalid timeout",
			cfg: &Config{
				TimeoutConfig: exporterhelper.TimeoutConfig{
					Timeout: -1 * time.Second,
				},
				QueueConfig: exporterhelper.NewDefaultQueueConfig(),
				RetryConfig: configretry.NewDefaultBackOffConfig(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
