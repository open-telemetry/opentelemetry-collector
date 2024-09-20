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
	"go.opentelemetry.io/collector/internal/testutil"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	require.NoError(t, componenttest.CheckConfigStruct(cfg))
	ocfg, ok := factory.CreateDefaultConfig().(*Config)
	assert.True(t, ok)
	assert.Equal(t, "", ocfg.ClientConfig.Endpoint)
	assert.Equal(t, 30*time.Second, ocfg.ClientConfig.Timeout, "default timeout is 30 second")
	assert.True(t, ocfg.RetryConfig.Enabled, "default retry is enabled")
	assert.Equal(t, 300*time.Second, ocfg.RetryConfig.MaxElapsedTime, "default retry MaxElapsedTime")
	assert.Equal(t, 5*time.Second, ocfg.RetryConfig.InitialInterval, "default retry InitialInterval")
	assert.Equal(t, 30*time.Second, ocfg.RetryConfig.MaxInterval, "default retry MaxInterval")
	assert.True(t, ocfg.QueueConfig.Enabled, "default sending queue is enabled")
	assert.Equal(t, EncodingProto, ocfg.Encoding)
	assert.Equal(t, configcompression.TypeGzip, ocfg.Compression)
}

func TestCreateMetricsExporter(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.Endpoint = "http://" + testutil.GetAvailableLocalAddress(t)

	set := exportertest.NewNopSettings()
	oexp, err := factory.CreateMetricsExporter(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, oexp)
}

func TestCreateTracesExporter(t *testing.T) {
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
				ClientConfig: confighttp.ClientConfig{
					Endpoint: "",
				},
			},
			mustFailOnCreate: true,
		},
		{
			name: "UseSecure",
			config: &Config{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: endpoint,
					TLSSetting: configtls.ClientConfig{
						Insecure: false,
					},
				},
			},
		},
		{
			name: "Headers",
			config: &Config{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: endpoint,
					Headers: map[string]configopaque.String{
						"hdr1": "val1",
						"hdr2": "val2",
					},
				},
			},
		},
		{
			name: "CaCert",
			config: &Config{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: endpoint,
					TLSSetting: configtls.ClientConfig{
						Config: configtls.Config{
							CAFile: filepath.Join("testdata", "test_cert.pem"),
						},
					},
				},
			},
		},
		{
			name: "CertPemFileError",
			config: &Config{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: endpoint,
					TLSSetting: configtls.ClientConfig{
						Config: configtls.Config{
							CAFile: "nosuchfile",
						},
					},
				},
			},
			mustFailOnCreate: false,
			mustFailOnStart:  true,
		},
		{
			name: "NoneCompression",
			config: &Config{
				ClientConfig: confighttp.ClientConfig{
					Endpoint:    endpoint,
					Compression: "none",
				},
			},
		},
		{
			name: "GzipCompression",
			config: &Config{
				ClientConfig: confighttp.ClientConfig{
					Endpoint:    endpoint,
					Compression: configcompression.TypeGzip,
				},
			},
		},
		{
			name: "SnappyCompression",
			config: &Config{
				ClientConfig: confighttp.ClientConfig{
					Endpoint:    endpoint,
					Compression: configcompression.TypeSnappy,
				},
			},
		},
		{
			name: "ZstdCompression",
			config: &Config{
				ClientConfig: confighttp.ClientConfig{
					Endpoint:    endpoint,
					Compression: configcompression.TypeZstd,
				},
			},
		},
		{
			name: "ProtoEncoding",
			config: &Config{
				Encoding:     EncodingProto,
				ClientConfig: confighttp.ClientConfig{Endpoint: endpoint},
			},
		},
		{
			name: "JSONEncoding",
			config: &Config{
				Encoding:     EncodingJSON,
				ClientConfig: confighttp.ClientConfig{Endpoint: endpoint},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory()
			set := exportertest.NewNopSettings()
			consumer, err := factory.CreateTracesExporter(context.Background(), set, tt.config)

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

func TestCreateLogsExporter(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.Endpoint = "http://" + testutil.GetAvailableLocalAddress(t)

	set := exportertest.NewNopSettings()
	oexp, err := factory.CreateLogsExporter(context.Background(), set, cfg)
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
}
