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
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))
	ocfg, ok := factory.CreateDefaultConfig().(*Config)
	assert.True(t, ok)
	assert.Equal(t, ocfg.HTTPClientSettings.Endpoint, "")
	assert.Equal(t, ocfg.HTTPClientSettings.Timeout, 30*time.Second, "default timeout is 30 second")
	assert.Equal(t, ocfg.RetrySettings.Enabled, true, "default retry is enabled")
	assert.Equal(t, ocfg.RetrySettings.MaxElapsedTime, 300*time.Second, "default retry MaxElapsedTime")
	assert.Equal(t, ocfg.RetrySettings.InitialInterval, 5*time.Second, "default retry InitialInterval")
	assert.Equal(t, ocfg.RetrySettings.MaxInterval, 30*time.Second, "default retry MaxInterval")
	assert.Equal(t, ocfg.QueueSettings.Enabled, true, "default sending queue is enabled")
	assert.Equal(t, ocfg.Compression, configcompression.Gzip)
}

func TestCreateMetricsExporter(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.HTTPClientSettings.Endpoint = "http://" + testutil.GetAvailableLocalAddress(t)

	set := exportertest.NewNopCreateSettings()
	oexp, err := factory.CreateMetricsExporter(context.Background(), set, cfg)
	require.Nil(t, err)
	require.NotNil(t, oexp)
}

func TestCreateTracesExporter(t *testing.T) {
	endpoint := "http://" + testutil.GetAvailableLocalAddress(t)

	tests := []struct {
		name             string
		config           Config
		mustFailOnCreate bool
		mustFailOnStart  bool
	}{
		{
			name: "NoEndpoint",
			config: Config{
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint: "",
				},
			},
			mustFailOnCreate: true,
		},
		{
			name: "UseSecure",
			config: Config{
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint: endpoint,
					TLSSetting: configtls.TLSClientSetting{
						Insecure: false,
					},
				},
			},
		},
		{
			name: "Headers",
			config: Config{
				HTTPClientSettings: confighttp.HTTPClientSettings{
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
			config: Config{
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint: endpoint,
					TLSSetting: configtls.TLSClientSetting{
						TLSSetting: configtls.TLSSetting{
							CAFile: filepath.Join("testdata", "test_cert.pem"),
						},
					},
				},
			},
		},
		{
			name: "CertPemFileError",
			config: Config{
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint: endpoint,
					TLSSetting: configtls.TLSClientSetting{
						TLSSetting: configtls.TLSSetting{
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
			config: Config{
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint:    endpoint,
					Compression: "none",
				},
			},
		},
		{
			name: "GzipCompression",
			config: Config{
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint:    endpoint,
					Compression: configcompression.Gzip,
				},
			},
		},
		{
			name: "SnappyCompression",
			config: Config{
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint:    endpoint,
					Compression: configcompression.Snappy,
				},
			},
		},
		{
			name: "ZstdCompression",
			config: Config{
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint:    endpoint,
					Compression: configcompression.Zstd,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory()
			set := exportertest.NewNopCreateSettings()
			consumer, err := factory.CreateTracesExporter(context.Background(), set, &tt.config)

			if tt.mustFailOnCreate {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, consumer)
			err = consumer.Start(context.Background(), componenttest.NewNopHost())
			if tt.mustFailOnStart {
				assert.Error(t, err)
			}

			err = consumer.Shutdown(context.Background())
			if err != nil {
				// Since the endpoint of OTLP exporter doesn't actually exist,
				// exporter may already stop because it cannot connect.
				assert.Equal(t, err.Error(), "rpc error: code = Canceled desc = grpc: the client connection is closing")
			}
		})
	}
}

func TestCreateLogsExporter(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.HTTPClientSettings.Endpoint = "http://" + testutil.GetAvailableLocalAddress(t)

	set := exportertest.NewNopCreateSettings()
	oexp, err := factory.CreateLogsExporter(context.Background(), set, cfg)
	require.Nil(t, err)
	require.NotNil(t, oexp)
}

func TestComposeSignalURL(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)

	// Has slash at end
	cfg.HTTPClientSettings.Endpoint = "http://localhost:4318/"
	url, err := composeSignalURL(cfg, "", "traces")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:4318/v1/traces", url)

	// No slash at end
	cfg.HTTPClientSettings.Endpoint = "http://localhost:4318"
	url, err = composeSignalURL(cfg, "", "traces")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:4318/v1/traces", url)
}
