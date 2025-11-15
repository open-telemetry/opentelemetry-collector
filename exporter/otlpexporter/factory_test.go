// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpexporter

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
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
	assert.Equal(t, configretry.NewDefaultBackOffConfig(), ocfg.RetryConfig)
	assert.Equal(t, exporterhelper.NewDefaultQueueConfig(), ocfg.QueueConfig)
	assert.Equal(t, exporterhelper.NewDefaultTimeoutConfig(), ocfg.TimeoutConfig)
	assert.Equal(t, configcompression.TypeGzip, ocfg.ClientConfig.Compression)
}

func TestCreateMetrics(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.Endpoint = testutil.GetAvailableLocalAddress(t)

	set := exportertest.NewNopSettings(factory.Type())
	oexp, err := factory.CreateMetrics(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, oexp)
}

func TestCreateTraces(t *testing.T) {
	endpoint := testutil.GetAvailableLocalAddress(t)
	tests := []struct {
		name            string
		config          *Config
		mustFailOnStart bool
	}{
		{
			name: "UseSecure",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint: endpoint,
					TLS: configtls.ClientConfig{
						Insecure: false,
					},
				},
			},
		},
		{
			name: "Keepalive",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint: endpoint,
					Keepalive: configoptional.Some(configgrpc.KeepaliveClientConfig{
						Time:                30 * time.Second,
						Timeout:             25 * time.Second,
						PermitWithoutStream: true,
					}),
				},
			},
		},
		{
			name: "NoneCompression",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint:    endpoint,
					Compression: "none",
				},
			},
		},
		{
			name: "GzipCompression",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint:    endpoint,
					Compression: configcompression.TypeGzip,
				},
			},
		},
		{
			name: "SnappyCompression",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint:    endpoint,
					Compression: configcompression.TypeSnappy,
				},
			},
		},
		{
			name: "ZstdCompression",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint:    endpoint,
					Compression: configcompression.TypeZstd,
				},
			},
		},
		{
			name: "Headers",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint: endpoint,
					Headers: configopaque.MapList{
						{Name: "hdr1", Value: "val1"},
						{Name: "hdr2", Value: "val2"},
					},
				},
			},
		},
		{
			name: "NumConsumers",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint: endpoint,
				},
			},
		},
		{
			name: "CaCert",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint: endpoint,
					TLS: configtls.ClientConfig{
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
				ClientConfig: configgrpc.ClientConfig{
					Endpoint: endpoint,
					TLS: configtls.ClientConfig{
						Config: configtls.Config{
							CAFile: "nosuchfile",
						},
					},
				},
			},
			mustFailOnStart: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory()
			set := exportertest.NewNopSettings(factory.Type())
			consumer, err := factory.CreateTraces(context.Background(), set, tt.config)
			require.NoError(t, err)
			assert.NotNil(t, consumer)
			err = consumer.Start(context.Background(), componenttest.NewNopHost())
			if tt.mustFailOnStart {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			// Shutdown is called even when Start fails
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
	cfg.ClientConfig.Endpoint = testutil.GetAvailableLocalAddress(t)

	set := exportertest.NewNopSettings(factory.Type())
	oexp, err := factory.CreateLogs(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, oexp)
}

func TestCreateProfiles(t *testing.T) {
	endpoint := testutil.GetAvailableLocalAddress(t)
	tests := []struct {
		name            string
		config          *Config
		mustFailOnStart bool
	}{
		{
			name: "UseSecure",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint: endpoint,
					TLS: configtls.ClientConfig{
						Insecure: false,
					},
				},
			},
		},
		{
			name: "Keepalive",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint: endpoint,
					Keepalive: configoptional.Some(configgrpc.KeepaliveClientConfig{
						Time:                30 * time.Second,
						Timeout:             25 * time.Second,
						PermitWithoutStream: true,
					}),
				},
			},
		},
		{
			name: "NoneCompression",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint:    endpoint,
					Compression: "none",
				},
			},
		},
		{
			name: "GzipCompression",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint:    endpoint,
					Compression: configcompression.TypeGzip,
				},
			},
		},
		{
			name: "SnappyCompression",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint:    endpoint,
					Compression: configcompression.TypeSnappy,
				},
			},
		},
		{
			name: "ZstdCompression",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint:    endpoint,
					Compression: configcompression.TypeZstd,
				},
			},
		},
		{
			name: "Headers",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint: endpoint,
					Headers: configopaque.MapList{
						{Name: "hdr1", Value: "val1"},
						{Name: "hdr2", Value: "val2"},
					},
				},
			},
		},
		{
			name: "NumConsumers",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint: endpoint,
				},
			},
		},
		{
			name: "CaCert",
			config: &Config{
				ClientConfig: configgrpc.ClientConfig{
					Endpoint: endpoint,
					TLS: configtls.ClientConfig{
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
				ClientConfig: configgrpc.ClientConfig{
					Endpoint: endpoint,
					TLS: configtls.ClientConfig{
						Config: configtls.Config{
							CAFile: "nosuchfile",
						},
					},
				},
			},
			mustFailOnStart: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory()
			set := exportertest.NewNopSettings(factory.Type())
			consumer, err := factory.(xexporter.Factory).CreateProfiles(context.Background(), set, tt.config)
			require.NoError(t, err)
			assert.NotNil(t, consumer)
			err = consumer.Start(context.Background(), componenttest.NewNopHost())
			if tt.mustFailOnStart {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			// Shutdown is called even when Start fails
			err = consumer.Shutdown(context.Background())
			if err != nil {
				// Since the endpoint of OTLP exporter doesn't actually exist,
				// exporter may already stop because it cannot connect.
				assert.Equal(t, "rpc error: code = Canceled desc = grpc: the client connection is closing", err.Error())
			}
		})
	}
}

func TestNewFactoryWithAlias(t *testing.T) {
	factory := NewFactoryWithAlias()
	assert.NotNil(t, factory)

	// Verify the factory type is otlp_grpc
	assert.Equal(t, component.MustNewType("otlp_grpc"), factory.Type())

	// Verify it can create default config
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	require.NoError(t, componenttest.CheckConfigStruct(cfg))

	// Verify stability levels match the original factory
	originalFactory := NewFactory()
	assert.Equal(t, originalFactory.TracesStability(), factory.TracesStability())
	assert.Equal(t, originalFactory.MetricsStability(), factory.MetricsStability())
	assert.Equal(t, originalFactory.LogsStability(), factory.LogsStability())
}

func TestAliasFactoryCreateMetrics(t *testing.T) {
	factory := NewFactoryWithAlias()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.Endpoint = testutil.GetAvailableLocalAddress(t)

	// Use the alias type for settings
	set := exportertest.NewNopSettings(factory.Type())
	oexp, err := factory.CreateMetrics(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, oexp)
}

func TestAliasFactoryCreateTraces(t *testing.T) {
	factory := NewFactoryWithAlias()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.Endpoint = testutil.GetAvailableLocalAddress(t)

	// Use the alias type for settings
	set := exportertest.NewNopSettings(factory.Type())
	oexp, err := factory.CreateTraces(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, oexp)
}

func TestAliasFactoryCreateLogs(t *testing.T) {
	factory := NewFactoryWithAlias()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.Endpoint = testutil.GetAvailableLocalAddress(t)

	// Use the alias type for settings
	set := exportertest.NewNopSettings(factory.Type())
	oexp, err := factory.CreateLogs(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, oexp)
}

func TestAliasFactoryCreateProfiles(t *testing.T) {
	factory := NewFactoryWithAlias()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.Endpoint = testutil.GetAvailableLocalAddress(t)

	// Use the alias type for settings
	set := exportertest.NewNopSettings(factory.Type())
	xFactory, ok := factory.(xexporter.Factory)
	require.True(t, ok)
	oexp, err := xFactory.CreateProfiles(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, oexp)
}

func TestAliasFactoryWithNamedComponent(t *testing.T) {
	factory := NewFactoryWithAlias()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.Endpoint = testutil.GetAvailableLocalAddress(t)

	// Test with a named component (e.g., otlp_grpc/myname)
	aliasType := component.MustNewType("otlp_grpc")
	componentID := component.NewIDWithName(aliasType, "myname")
	set := exportertest.NewNopSettings(componentID.Type())
	set.ID = componentID

	oexp, err := factory.CreateMetrics(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, oexp)
}
