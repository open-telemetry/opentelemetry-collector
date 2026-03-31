// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlphttpexporter

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/xexporter"
	"go.opentelemetry.io/collector/internal/testutil"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	require.NoError(t, componenttest.CheckConfigStruct(cfg))
	ocfg, ok := factory.CreateDefaultConfig().(*Config)
	assert.True(t, ok)
	assert.Empty(t, ocfg.ClientConfig.Endpoint)
	assert.Equal(t, 30*time.Second, ocfg.ClientConfig.Timeout, "default timeout is 30 second")
	assert.True(t, ocfg.RetryConfig.Enabled, "default retry is enabled")
	assert.Equal(t, 300*time.Second, ocfg.RetryConfig.MaxElapsedTime, "default retry MaxElapsedTime")
	assert.Equal(t, 5*time.Second, ocfg.RetryConfig.InitialInterval, "default retry InitialInterval")
	assert.Equal(t, 30*time.Second, ocfg.RetryConfig.MaxInterval, "default retry MaxInterval")
	assert.True(t, ocfg.QueueConfig.HasValue(), "default sending queue is enabled")
	assert.Equal(t, EncodingProto, ocfg.Encoding)
	assert.Equal(t, configcompression.TypeGzip, ocfg.ClientConfig.Compression)
}

func TestCreateMetrics(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.Endpoint = "http://" + testutil.GetAvailableLocalAddress(t)

	set := exportertest.NewNopSettings(factory.Type())
	oexp, err := factory.CreateMetrics(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, oexp)
}

func clientConfig(endpoint string, headers configopaque.MapList, tlsSetting configtls.ClientConfig, compression configcompression.Type) confighttp.ClientConfig {
	clientConfig := confighttp.NewDefaultClientConfig()
	clientConfig.TLS = tlsSetting
	clientConfig.Compression = compression
	if endpoint != "" {
		clientConfig.Endpoint = endpoint
	}
	if headers != nil {
		clientConfig.Headers = headers
	}
	return clientConfig
}

func TestCreateTraces(t *testing.T) {
	var configCompression configcompression.Type
	endpoint := "http://" + testutil.GetAvailableLocalAddress(t)

	tests := []struct {
		name             string
		config           *Config
		mustFailOnCreate bool
		mustFailOnStart  bool
	}{
		{
			name: "NoEndpoint",
			config: &Config{
				ClientConfig: clientConfig("", nil, configtls.ClientConfig{}, configCompression),
			},
			mustFailOnCreate: true,
		},
		{
			name: "UseSecure",
			config: &Config{
				ClientConfig: clientConfig(endpoint, nil, configtls.ClientConfig{
					Insecure: false,
				}, configCompression),
			},
		},
		{
			name: "Headers",
			config: &Config{
				ClientConfig: clientConfig(endpoint, configopaque.MapList{
					{Name: "hdr1", Value: "val1"},
					{Name: "hdr2", Value: "val2"},
				}, configtls.ClientConfig{}, configCompression),
			},
		},
		{
			name: "CaCert",
			config: &Config{
				ClientConfig: clientConfig(endpoint, nil, configtls.ClientConfig{
					Config: configtls.Config{
						CAFile: filepath.Join("testdata", "test_cert.pem"),
					},
				}, configCompression),
			},
		},
		{
			name: "CertPemFileError",
			config: &Config{
				ClientConfig: clientConfig(endpoint, nil, configtls.ClientConfig{
					Config: configtls.Config{
						CAFile: "nosuchfile",
					},
				},
					configCompression),
			},
			mustFailOnCreate: false,
			mustFailOnStart:  true,
		},
		{
			name: "NoneCompression",
			config: &Config{
				ClientConfig: clientConfig(endpoint, nil, configtls.ClientConfig{}, "none"),
			},
		},
		{
			name: "GzipCompression",
			config: &Config{
				ClientConfig: clientConfig(endpoint, nil, configtls.ClientConfig{}, configcompression.TypeGzip),
			},
		},
		{
			name: "SnappyCompression",
			config: &Config{
				ClientConfig: clientConfig(endpoint, nil, configtls.ClientConfig{}, configcompression.TypeSnappy),
			},
		},
		{
			name: "ZstdCompression",
			config: &Config{
				ClientConfig: clientConfig(endpoint, nil, configtls.ClientConfig{}, configcompression.TypeZstd),
			},
		},
		{
			name: "ProtoEncoding",
			config: &Config{
				Encoding:     EncodingProto,
				ClientConfig: clientConfig(endpoint, nil, configtls.ClientConfig{}, configCompression),
			},
		},
		{
			name: "JSONEncoding",
			config: &Config{
				Encoding:     EncodingJSON,
				ClientConfig: clientConfig(endpoint, nil, configtls.ClientConfig{}, configCompression),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory()
			set := exportertest.NewNopSettings(factory.Type())
			consumer, err := factory.CreateTraces(context.Background(), set, tt.config)

			if tt.mustFailOnCreate {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, consumer)
			err = consumer.Start(context.Background(), componenttest.NewNopHost())
			if tt.mustFailOnStart {
				require.Error(t, err)
			}

			err = consumer.Shutdown(context.Background())
			if err != nil {
				// Since the endpoint of OTLP exporter doesn't actually exist,
				// exporter may already stop because it cannot connect.
				assert.Equal(t, "rpc error: code = Canceled desc = grpc: the client connection is closing", err.Error())
			}
		})
	}
}

func TestCreateLogs(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.Endpoint = "http://" + testutil.GetAvailableLocalAddress(t)

	set := exportertest.NewNopSettings(factory.Type())
	oexp, err := factory.CreateLogs(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, oexp)
}

func TestCreateProfiles(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.Endpoint = "http://" + testutil.GetAvailableLocalAddress(t)

	set := exportertest.NewNopSettings(factory.Type())
	oexp, err := factory.(xexporter.Factory).CreateProfiles(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, oexp)
}

func TestCreateProfilesWithCustomEndpoint(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ProfilesEndpoint = "http://" + testutil.GetAvailableLocalAddress(t) + "/custom/profiles"

	set := exportertest.NewNopSettings(factory.Type())
	oexp, err := factory.(xexporter.Factory).CreateProfiles(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, oexp)
}

func TestComposeSignalURL(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)

	// Has slash at end
	cfg.ClientConfig.Endpoint = "http://localhost:4318/"
	url, err := composeSignalURL(cfg, "", "traces", "v1")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:4318/v1/traces", url)

	// No slash at end
	cfg.ClientConfig.Endpoint = "http://localhost:4318"
	url, err = composeSignalURL(cfg, "", "traces", "v1")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:4318/v1/traces", url)

	// Different version
	cfg.ClientConfig.Endpoint = "http://localhost:4318"
	url, err = composeSignalURL(cfg, "", "traces", "v2")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:4318/v2/traces", url)

	// Test profiles endpoint with v1development
	cfg.ClientConfig.Endpoint = "http://localhost:4318"
	url, err = composeSignalURL(cfg, "", "profiles", "v1development")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:4318/v1development/profiles", url)

	// Test with custom profiles endpoint override
	cfg.ClientConfig.Endpoint = "http://localhost:4318"
	url, err = composeSignalURL(cfg, "http://custom:9090/profiles", "profiles", "v1development")
	require.NoError(t, err)
	assert.Equal(t, "http://custom:9090/profiles", url)
}
