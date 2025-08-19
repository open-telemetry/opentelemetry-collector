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
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

func TestUnmarshalDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NoError(t, confmap.New().Unmarshal(&cfg))
	assert.Equal(t, factory.CreateDefaultConfig(), cfg)
}

func TestUnmarshalConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NoError(t, cm.Unmarshal(&cfg))
	require.NoError(t, xconfmap.Validate(&cfg))
	assert.Equal(t,
		&Config{
			TimeoutConfig: exporterhelper.TimeoutConfig{
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
			QueueConfig: exporterhelper.QueueBatchConfig{
				Enabled:      true,
				Sizer:        exporterhelper.RequestSizerTypeItems,
				NumConsumers: 2,
				QueueSize:    100000,
				Batch: configoptional.Some(exporterhelper.BatchConfig{
					FlushTimeout: 200 * time.Millisecond,
					Sizer:        exporterhelper.RequestSizerTypeItems,
					MinSize:      1000,
					MaxSize:      10000,
				}),
			},
			ClientConfig: configgrpc.ClientConfig{
				Headers: map[string]configopaque.String{
					"can you have a . here?": "F0000000-0000-0000-0000-000000000000",
					"header1":                "234",
					"another":                "somevalue",
				},
				Endpoint:    "1.2.3.4:1234",
				Compression: "gzip",
				TLS: configtls.ClientConfig{
					Config: configtls.Config{
						CAFile: "/var/lib/mycert.pem",
					},
					Insecure: false,
				},
				Keepalive: configoptional.Some(configgrpc.KeepaliveClientConfig{
					Time:                20 * time.Second,
					PermitWithoutStream: true,
					Timeout:             30 * time.Second,
				}),
				WriteBufferSize: 512 * 1024,
				BalancerName:    "round_robin",
				Auth:            configoptional.Some(configauth.Config{AuthenticatorID: component.MustNewID("nop")}),
			},
		}, cfg)
}

func TestUnmarshalInvalidConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "invalid_configs.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	for _, tt := range []struct {
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
		t.Run(tt.name, func(t *testing.T) {
			cfg := factory.CreateDefaultConfig()
			sub, err := cm.Sub(tt.name)
			require.NoError(t, err)
			assert.NoError(t, sub.Unmarshal(&cfg))
			assert.ErrorContains(t, xconfmap.Validate(cfg), tt.errorMsg)
		})
	}
}

func TestValidDNSEndpoint(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.Endpoint = "dns://authority/backend.example.com:4317"
	assert.NoError(t, cfg.Validate())
}

func TestValid(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)

	testCases := []struct {
		endpoint      string
		expectedError bool
	}{
		{"dns://authority/backend.example.com:4317", false},
		{"dns:///backend.example.com:4317", false},
		{"http://backend.example.com:4317", false},
		{"https://backend.example.com:4317", false},
		{"backend.example.com:4317", false},
		{"unix:///tmp/otlp.sock", false},
		{"unix:@abstract-socket", false},
		{"dns:////backend.example.com:4317", false},

		// ========= invalid cases =========
		{"unix:/tmp/otlp.sock", true},
		{"ftp://backend.example.com:21", true},    // Unknow schema
		{"dns:///backend.example.com", true},      // no port
		{"dns:///backend.example.com:abcd", true}, // no number port
		{"backend.example.com", true},             // no port
		{"backend.example.com:notaport", true},    // no number port
		{"http://backend.example.com", true},      // http no port
		{"https://backend.example.com", true},     // https no port
		{"", true},                                // null endpoint

		// ========= special cases =========
		{"unix", true},
		{"unix:", true},
		{"unix://", true},

		{"http:", true},
		{"http://", true},
		{"http:///", true},

		{"https:", true},
		{"https://", true},
		{"https:///", true},

		{"dns:", true},
		{"dns://", true},
		{"dns:///", true},
	}

	for _, tc := range testCases {
		t.Run(tc.endpoint, func(t *testing.T) {
			cfg.ClientConfig.Endpoint = tc.endpoint
			err := cfg.Validate()
			if tc.expectedError {
				assert.Errorf(t, err, "expected error but got none err: %v", err)
			} else {
				assert.NoErrorf(t, err, "expected no error but got one: %v", err)
			}
		})
	}
}

func TestHostPortSanitizeEndpoint(t *testing.T) {
	testCases := []struct {
		endpoint         string
		prefix           string
		expectedSanitize string
	}{
		// ===== DNS =====
		{"dns://authority/backend.example.com:4317", "dns://", "authority/backend.example.com:4317"},
		{"dns:///backend.example.com:4317", "dns://", "backend.example.com:4317"},
		{"dns:////backend.example.com:4317", "dns://", "/backend.example.com:4317"},
		{"dns:///backend.example.com", "dns://", "backend.example.com"},

		// ===== HTTP =====
		{"http://backend.example.com:4317", "http://", "backend.example.com:4317"},
		{"http://backend.example.com", "http://", "backend.example.com"},

		// ===== HTTPS =====
		{"https://backend.example.com:4317", "https://", "backend.example.com:4317"},
		{"https://backend.example.com", "https://", "backend.example.com"},

		// ===== Null prefix(host:port)  =====
		{"backend.example.com:4317", "", "backend.example.com:4317"},
		{"backend.example.com", "", "backend.example.com"},
	}

	for _, tc := range testCases {
		v := HostPortValidator{prefix: tc.prefix}
		t.Run(tc.endpoint, func(t *testing.T) {
			sanitized := v.Sanitize(tc.endpoint)
			assert.Equal(t, tc.expectedSanitize, sanitized)
		})
	}
}
