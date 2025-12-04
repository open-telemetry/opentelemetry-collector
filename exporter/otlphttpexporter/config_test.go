// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlphttpexporter

import (
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/confighttp"
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
	// Default/Empty config is invalid.
	assert.Error(t, xconfmap.Validate(cfg))
}

func TestUnmarshalConfig(t *testing.T) {
	defaultMaxIdleConns := http.DefaultTransport.(*http.Transport).MaxIdleConns
	defaultMaxIdleConnsPerHost := http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost
	defaultMaxConnsPerHost := http.DefaultTransport.(*http.Transport).MaxConnsPerHost
	defaultIdleConnTimeout := http.DefaultTransport.(*http.Transport).IdleConnTimeout

	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NoError(t, cm.Unmarshal(&cfg))
	assert.Equal(t,
		&Config{
			RetryConfig: configretry.BackOffConfig{
				Enabled:             true,
				InitialInterval:     10 * time.Second,
				RandomizationFactor: 0.7,
				Multiplier:          1.3,
				MaxInterval:         1 * time.Minute,
				MaxElapsedTime:      10 * time.Minute,
			},
			QueueConfig: configoptional.Some(exporterhelper.QueueBatchConfig{
				Sizer:        exporterhelper.RequestSizerTypeRequests,
				NumConsumers: 2,
				QueueSize:    10,
				Batch: configoptional.Default(exporterhelper.BatchConfig{
					Sizer:        exporterhelper.RequestSizerTypeItems,
					FlushTimeout: 200 * time.Millisecond,
					MinSize:      8192,
				}),
			}),
			Encoding: EncodingProto,
			ClientConfig: confighttp.ClientConfig{
				Headers: configopaque.MapList{
					{Name: "another", Value: "somevalue"},
					{Name: "can you have a . here?", Value: "F0000000-0000-0000-0000-000000000000"},
					{Name: "header1", Value: "234"},
				},
				Endpoint: "https://1.2.3.4:1234",
				TLS: configtls.ClientConfig{
					Config: configtls.Config{
						CAFile:   "/var/lib/mycert.pem",
						CertFile: "certfile",
						KeyFile:  "keyfile",
					},
					Insecure: true,
				},
				ReadBufferSize:      123,
				WriteBufferSize:     345,
				Timeout:             time.Second * 10,
				Compression:         "gzip",
				MaxIdleConns:        defaultMaxIdleConns,
				MaxIdleConnsPerHost: defaultMaxIdleConnsPerHost,
				MaxConnsPerHost:     defaultMaxConnsPerHost,
				IdleConnTimeout:     defaultIdleConnTimeout,
				ForceAttemptHTTP2:   true,
			},
			ProfilesEndpoint: "https://custom.profiles.endpoint:8080/v1development/profiles",
		}, cfg)
}

func TestUnmarshalConfigInvalidEncoding(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "bad_invalid_encoding.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.Error(t, cm.Unmarshal(&cfg))
}

func TestUnmarshalEncoding(t *testing.T) {
	tests := []struct {
		name          string
		encodingBytes []byte
		expected      EncodingType
		shouldError   bool
	}{
		{
			name:          "UnmarshalEncodingProto",
			encodingBytes: []byte("proto"),
			expected:      EncodingProto,
			shouldError:   false,
		},
		{
			name:          "UnmarshalEncodingJson",
			encodingBytes: []byte("json"),
			expected:      EncodingJSON,
			shouldError:   false,
		},
		{
			name:          "UnmarshalEmptyEncoding",
			encodingBytes: []byte(""),
			shouldError:   true,
		},
		{
			name:          "UnmarshalInvalidEncoding",
			encodingBytes: []byte("invalid"),
			shouldError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var encoding EncodingType
			err := encoding.UnmarshalText(tt.encodingBytes)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, encoding)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "no endpoints specified",
			cfg: &Config{
				ClientConfig: confighttp.ClientConfig{},
			},
			wantErr: true,
		},
		{
			name: "main endpoint specified",
			cfg: &Config{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: "http://localhost:4318",
				},
			},
			wantErr: false,
		},
		{
			name: "only traces endpoint specified",
			cfg: &Config{
				ClientConfig:   confighttp.ClientConfig{},
				TracesEndpoint: "http://localhost:4318/v1/traces",
			},
			wantErr: false,
		},
		{
			name: "only profiles endpoint specified",
			cfg: &Config{
				ClientConfig:     confighttp.ClientConfig{},
				ProfilesEndpoint: "http://localhost:4318/v1development/profiles",
			},
			wantErr: false,
		},
		{
			name: "multiple endpoints specified",
			cfg: &Config{
				ClientConfig:     confighttp.ClientConfig{},
				TracesEndpoint:   "http://localhost:4318/v1/traces",
				MetricsEndpoint:  "http://localhost:4318/v1/metrics",
				LogsEndpoint:     "http://localhost:4318/v1/logs",
				ProfilesEndpoint: "http://localhost:4318/v1development/profiles",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "at least one endpoint must be specified")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
