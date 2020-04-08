// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/compression"
	"github.com/open-telemetry/opentelemetry-collector/config/configcheck"
	"github.com/open-telemetry/opentelemetry-collector/config/configgrpc"
	"github.com/open-telemetry/opentelemetry-collector/testutils"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateMetricsExporter(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCSettings.Endpoint = testutils.GetAvailableLocalAddress(t)

	creationParams := component.ExporterCreateParams{Logger: zap.NewNop()}
	oexp, err := factory.CreateMetricsExporter(context.Background(), creationParams, cfg)
	require.Nil(t, err)
	require.NotNil(t, oexp)
}

func TestCreateTraceExporter(t *testing.T) {
	endpoint := testutils.GetAvailableLocalAddress(t)

	tests := []struct {
		name     string
		config   Config
		mustFail bool
	}{
		{
			name: "NoEndpoint",
			config: Config{
				GRPCSettings: configgrpc.GRPCSettings{
					Endpoint: "",
				},
			},
			mustFail: true,
		},
		{
			name: "UseSecure",
			config: Config{
				GRPCSettings: configgrpc.GRPCSettings{
					Endpoint:  endpoint,
					UseSecure: true,
				},
			},
		},
		{
			name: "ReconnectionDelay",
			config: Config{
				GRPCSettings: configgrpc.GRPCSettings{
					Endpoint: endpoint,
				},
				ReconnectionDelay: 5 * time.Second,
			},
		},
		{
			name: "KeepaliveParameters",
			config: Config{
				GRPCSettings: configgrpc.GRPCSettings{
					Endpoint: endpoint,
					KeepaliveParameters: &configgrpc.KeepaliveConfig{
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
				GRPCSettings: configgrpc.GRPCSettings{
					Endpoint:    endpoint,
					Compression: compression.Gzip,
				},
			},
		},
		{
			name: "Headers",
			config: Config{
				GRPCSettings: configgrpc.GRPCSettings{
					Endpoint: endpoint,
					Headers: map[string]string{
						"hdr1": "val1",
						"hdr2": "val2",
					},
				},
			},
		},
		{
			name: "NumWorkers",
			config: Config{
				GRPCSettings: configgrpc.GRPCSettings{
					Endpoint: endpoint,
				},
				NumWorkers: 3,
			},
		},
		{
			name: "CompressionError",
			config: Config{
				GRPCSettings: configgrpc.GRPCSettings{
					Endpoint:    endpoint,
					Compression: "unknown compression",
				},
			},
			mustFail: true,
		},
		{
			name: "CertPemFile",
			config: Config{
				GRPCSettings: configgrpc.GRPCSettings{
					Endpoint:    endpoint,
					CertPemFile: "testdata/test_cert.pem",
				},
			},
		},
		{
			name: "CertPemFileError",
			config: Config{
				GRPCSettings: configgrpc.GRPCSettings{
					Endpoint:    endpoint,
					CertPemFile: "nosuchfile",
				},
			},
			mustFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &Factory{}
			creationParams := component.ExporterCreateParams{Logger: zap.NewNop()}
			consumer, err := factory.CreateTraceExporter(context.Background(), creationParams, &tt.config)

			if tt.mustFail {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, consumer)

				err = consumer.Shutdown(context.Background())
				if err != nil {
					// Since the endpoint of OTLP exporter doesn't actually exist,
					// exporter may already stop because it cannot connect.
					assert.Equal(t, err.Error(), "rpc error: code = Canceled desc = grpc: the client connection is closing")
				}
			}
		})
	}
}
