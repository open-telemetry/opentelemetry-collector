// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otlpexporter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/internal/testutil"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
	ocfg, ok := factory.CreateDefaultConfig().(*Config)
	assert.True(t, ok)
	assert.Equal(t, ocfg.RetrySettings, exporterhelper.DefaultRetrySettings())
	assert.Equal(t, ocfg.QueueSettings, exporterhelper.DefaultQueueSettings())
	assert.Equal(t, ocfg.TimeoutSettings, exporterhelper.DefaultTimeoutSettings())
}

func TestCreateMetricsExporter(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings.Endpoint = testutil.GetAvailableLocalAddress(t)

	set := componenttest.NewNopExporterCreateSettings()
	oexp, err := factory.CreateMetricsExporter(context.Background(), set, cfg)
	require.Nil(t, err)
	require.NotNil(t, oexp)
}

func TestCreateTracesExporter(t *testing.T) {
	endpoint := testutil.GetAvailableLocalAddress(t)
	tests := []struct {
		name             string
		config           Config
		mustFailOnCreate bool
		mustFailOnStart  bool
	}{
		{
			name: "NoEndpoint",
			config: Config{
				ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: "",
				},
			},
			mustFailOnCreate: true,
		},
		{
			name: "UseSecure",
			config: Config{
				ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: endpoint,
					TLSSetting: configtls.TLSClientSetting{
						Insecure: false,
					},
				},
			},
		},
		{
			name: "Keepalive",
			config: Config{
				ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: endpoint,
					Keepalive: &configgrpc.KeepaliveClientConfig{
						Time:                30 * time.Second,
						Timeout:             25 * time.Second,
						PermitWithoutStream: true,
					},
				},
			},
		},
		{
			name: "Compression",
			config: Config{
				ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint:    endpoint,
					Compression: configgrpc.CompressionGzip,
				},
			},
		},
		{
			name: "Headers",
			config: Config{
				ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: endpoint,
					Headers: map[string]string{
						"hdr1": "val1",
						"hdr2": "val2",
					},
				},
			},
		},
		{
			name: "NumConsumers",
			config: Config{
				ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: endpoint,
				},
			},
		},
		{
			name: "CompressionError",
			config: Config{
				ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint:    endpoint,
					Compression: "unknown compression",
				},
			},
			mustFailOnStart: true,
		},
		{
			name: "CaCert",
			config: Config{
				ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: endpoint,
					TLSSetting: configtls.TLSClientSetting{
						TLSSetting: configtls.TLSSetting{
							CAFile: "testdata/test_cert.pem",
						},
					},
				},
			},
		},
		{
			name: "CertPemFileError",
			config: Config{
				ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: endpoint,
					TLSSetting: configtls.TLSClientSetting{
						TLSSetting: configtls.TLSSetting{
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
			set := componenttest.NewNopExporterCreateSettings()
			consumer, err := factory.CreateTracesExporter(context.Background(), set, &tt.config)
			if tt.mustFailOnCreate {
				assert.NotNil(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, consumer)
			err = consumer.Start(context.Background(), componenttest.NewNopHost())
			if tt.mustFailOnStart {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
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
	cfg.GRPCClientSettings.Endpoint = testutil.GetAvailableLocalAddress(t)

	set := componenttest.NewNopExporterCreateSettings()
	oexp, err := factory.CreateLogsExporter(context.Background(), set, cfg)
	require.Nil(t, err)
	require.NotNil(t, oexp)
}
