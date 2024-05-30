// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpexporter

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

func TestUnmarshalDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, confmap.New().Unmarshal(&cfg))
	assert.Equal(t, factory.CreateDefaultConfig(), cfg)
}

func TestUnmarshalConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, cm.Unmarshal(&cfg))
	assert.Equal(t,
		&Config{
			TimeoutSettings: exporterhelper.TimeoutSettings{
				Timeout: 10 * time.Second,
			},
			RetryConfig: configretry.BackOffConfig{
				Enabled:             true,
				InitialInterval:     10 * time.Second,
				RandomizationFactor: 0.7,
				Multiplier:          1.3,
				MaxInterval:         1 * time.Minute,
				MaxElapsedTime:      10 * time.Minute,
			},
			QueueConfig: exporterhelper.QueueSettings{
				Enabled:      true,
				NumConsumers: 2,
				QueueSize:    10,
			},
			ClientConfig: configgrpc.ClientConfig{
				Headers: map[string]configopaque.String{
					"can you have a . here?": "F0000000-0000-0000-0000-000000000000",
					"header1":                "234",
					"another":                "somevalue",
				},
				Endpoint:    "1.2.3.4:1234",
				Compression: "gzip",
				TLSSetting: configtls.ClientConfig{
					Config: configtls.Config{
						CAFile: "/var/lib/mycert.pem",
					},
					Insecure: false,
				},
				Keepalive: &configgrpc.KeepaliveClientConfig{
					Time:                20 * time.Second,
					PermitWithoutStream: true,
					Timeout:             30 * time.Second,
				},
				WriteBufferSize: 512 * 1024,
				BalancerName:    "round_robin",
				Auth:            &configauth.Authentication{AuthenticatorID: component.MustNewID("nop")},
			},
		}, cfg)
}

func TestUnmarshalInvalidConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "invalid_configs.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	for _, test := range []struct {
		name     string
		errorMsg string
	}{
		{
			name:     "no_endpoint",
			errorMsg: `requires a non-empty "endpoint"`,
		},
		{
			name:     "https_endpoint",
			errorMsg: `requires a non-empty "endpoint"`,
		},
		{
			name:     "http_endpoint",
			errorMsg: `requires a non-empty "endpoint"`,
		},
		{
			name:     "invalid_timeout",
			errorMsg: `'timeout' must be non-negative`,
		},
		{
			name:     "invalid_retry",
			errorMsg: `'randomization_factor' must be within [0, 1]`,
		},
		{
			name:     "invalid_tls",
			errorMsg: `invalid TLS min_version: unsupported TLS version: "asd"`,
		},
		{
			name:     "missing_port",
			errorMsg: `missing port in address`,
		},
		{
			name:     "invalid_port",
			errorMsg: `invalid port "port"`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := factory.CreateDefaultConfig()
			sub, err := cm.Sub(test.name)
			require.NoError(t, err)
			assert.NoError(t, sub.Unmarshal(&cfg))
			assert.ErrorContains(t, component.ValidateConfig(cfg), test.errorMsg)
		})
	}

}

func TestValidDNSEndpoint(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Endpoint = "dns:///backend.example.com:4317"
	assert.NoError(t, cfg.Validate())
}
