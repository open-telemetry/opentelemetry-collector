// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlphttpexporter

import (
	"fmt"
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
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

func TestUnmarshalDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NoError(t, confmap.New().Unmarshal(&cfg))
	assert.Equal(t, factory.CreateDefaultConfig(), cfg)
	// Default/Empty config is invalid.
	assert.Error(t, confmap.Validate(cfg))
}

func TestUnmarshalConfig(t *testing.T) {
	defaultMaxConnsPerHost := http.DefaultTransport.(*http.Transport).MaxConnsPerHost

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
				Batch:        configoptional.Default(exporterhelper.NewDefaultBatchConfig()),
			}),
			Encoding: EncodingProto,
			ClientConfig: confighttp.ClientConfig{
				Headers: configopaque.MapList{
					{Name: "another", Value: "somevalue"},
					{Name: "can you have a . here?", Value: "F0000000-0000-0000-0000-000000000000"},
					{Name: "header1", Value: "234"},
				},
				Endpoint: "https://1.2.3.4:1234",
				Keepalive: configoptional.Some(confighttp.KeepaliveClientConfig{
					MaxIdleConns:    100,
					IdleConnTimeout: 90 * time.Second,
				}),
				TLS: configtls.ClientConfig{
					Config: configtls.Config{
						CAFile:   "/var/lib/mycert.pem",
						CertFile: "certfile",
						KeyFile:  "keyfile",
					},
					Insecure: true,
				},
				ReadBufferSize:    123,
				WriteBufferSize:   345,
				Timeout:           time.Second * 10,
				Compression:       "gzip",
				MaxConnsPerHost:   defaultMaxConnsPerHost,
				ForceAttemptHTTP2: true,
			},
			ProfilesEndpoint:         "https://custom.profiles.endpoint:8080/v1development/profiles",
			RetryableHTTPStatusCodes: []int{http.StatusRequestTimeout},
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

func TestConfigValidateRetryableHTTPStatusCodes(t *testing.T) {
	const (
		errDisabled = "retryable_http_status_codes cannot be configured when retry_on_failure is disabled"
		err400      = "retryable_http_status_codes cannot contain 400 (Bad Request)"
		errRange    = "retryable_http_status_codes must only contain HTTP status codes in the range 401-599, got %d"
	)
	for _, tt := range []struct {
		name         string
		codes        []int
		retryEnabled bool
		wantErr      string
	}{
		{name: "empty", retryEnabled: true},
		{name: "nil with retry disabled"},
		{name: "empty slice with retry disabled", codes: []int{}},
		{name: "configured with retry disabled", codes: []int{http.StatusRequestTimeout}, wantErr: errDisabled},
		{name: "valid 401-599", codes: []int{http.StatusUnauthorized, http.StatusRequestTimeout, http.StatusInternalServerError, 599}, retryEnabled: true},
		{name: "default retryable codes allowed", codes: []int{http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout}, retryEnabled: true},
		{name: "400 Bad Request rejected", codes: []int{http.StatusBadRequest}, retryEnabled: true, wantErr: err400},
		{name: "duplicate status code rejected", codes: []int{408, 408}, retryEnabled: true, wantErr: "duplicate status code 408 in retryable_http_status_codes"},
		{name: "below 401 rejected", codes: []int{http.StatusOK}, retryEnabled: true, wantErr: fmt.Sprintf(errRange, http.StatusOK)},
		{name: "399 boundary rejected", codes: []int{399}, retryEnabled: true, wantErr: fmt.Sprintf(errRange, 399)},
		{name: "600 boundary rejected", codes: []int{600}, retryEnabled: true, wantErr: fmt.Sprintf(errRange, 600)},
		{name: "negative code rejected", codes: []int{408, -1}, retryEnabled: true, wantErr: fmt.Sprintf(errRange, -1)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			retryCfg := configretry.NewDefaultBackOffConfig()
			retryCfg.Enabled = tt.retryEnabled
			cfg := &Config{
				ClientConfig:             confighttp.ClientConfig{Endpoint: "http://localhost:4318"},
				RetryConfig:              retryCfg,
				RetryableHTTPStatusCodes: tt.codes,
			}
			if tt.wantErr != "" {
				require.EqualError(t, cfg.Validate(), tt.wantErr)
				require.EqualError(t, confmap.Validate(cfg), tt.wantErr)
				return
			}
			require.NoError(t, cfg.Validate())
			require.NoError(t, confmap.Validate(cfg))
		})
	}

	t.Run("unmarshal disabled retry fails Validate", func(t *testing.T) {
		cm := confmap.NewFromStringMap(map[string]any{
			"endpoint":                    "http://localhost:4318",
			"retry_on_failure":            map[string]any{"enabled": false},
			"retryable_http_status_codes": []int{http.StatusRequestTimeout},
		})
		cfg := NewFactory().CreateDefaultConfig()
		require.NoError(t, cm.Unmarshal(&cfg))
		require.EqualError(t, confmap.Validate(cfg), errDisabled)
	})
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
