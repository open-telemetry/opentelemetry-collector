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

package opencensusexporter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/testutil"
)

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
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
				NumWorkers: 3,
			},
			mustFailOnCreate: true,
		},
		{
			name: "ZeroNumWorkers",
			config: Config{
				ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: endpoint,
					TLSSetting: configtls.TLSClientSetting{
						Insecure: false,
					},
				},
				NumWorkers: 0,
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
				NumWorkers: 3,
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
				NumWorkers: 3,
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
				NumWorkers: 3,
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
				NumWorkers: 3,
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
				NumWorkers: 3,
			},
			mustFailOnCreate: false,
			mustFailOnStart:  true,
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
				NumWorkers: 3,
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
				NumWorkers: 3,
			},
			mustFailOnCreate: false,
			mustFailOnStart:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := componenttest.NewNopExporterCreateSettings()
			tExporter, tErr := createTracesExporter(context.Background(), set, &tt.config)
			checkErrorsAndStartAndShutdown(t, tExporter, tErr, tt.mustFailOnCreate, tt.mustFailOnStart)
			mExporter, mErr := createMetricsExporter(context.Background(), set, &tt.config)
			checkErrorsAndStartAndShutdown(t, mExporter, mErr, tt.mustFailOnCreate, tt.mustFailOnStart)
		})
	}
}

func checkErrorsAndStartAndShutdown(t *testing.T, exporter component.Exporter, err error, mustFailOnCreate, mustFailOnStart bool) {
	if mustFailOnCreate {
		assert.NotNil(t, err)
		return
	}
	assert.NoError(t, err)
	assert.NotNil(t, exporter)

	sErr := exporter.Start(context.Background(), componenttest.NewNopHost())
	if mustFailOnStart {
		require.Error(t, sErr)
		return
	}
	require.NoError(t, sErr)
	require.NoError(t, exporter.Shutdown(context.Background()))
}
