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
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
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

func TestCreateTraceExporter(t *testing.T) {
	endpoint := testutil.GetAvailableLocalAddress(t)
	tests := []struct {
		name     string
		config   Config
		mustFail bool
	}{
		{
			name: "NoEndpoint",
			config: Config{
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: "",
				},
				NumWorkers: 3,
			},
			mustFail: true,
		},
		{
			name: "ZeroNumWorkers",
			config: Config{
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: endpoint,
					TLSSetting: configtls.TLSClientSetting{
						Insecure: false,
					},
				},
				NumWorkers: 0,
			},
			mustFail: true,
		},
		{
			name: "UseSecure",
			config: Config{
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
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint:    endpoint,
					Compression: "unknown compression",
				},
				NumWorkers: 3,
			},
			mustFail: true,
		},
		{
			name: "CaCert",
			config: Config{
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
			mustFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := component.ExporterCreateParams{Logger: zap.NewNop()}
			tReceiver, tErr := createTraceExporter(context.Background(), params, &tt.config)
			checkErrorsAndShutdown(t, tReceiver, tErr, tt.mustFail)
			mReceiver, mErr := createMetricsExporter(context.Background(), params, &tt.config)
			checkErrorsAndShutdown(t, mReceiver, mErr, tt.mustFail)
		})
	}
}

func checkErrorsAndShutdown(t *testing.T, receiver component.Receiver, err error, mustFail bool) {
	if mustFail {
		assert.NotNil(t, err)
	} else {
		assert.NoError(t, err)
		assert.NotNil(t, receiver)

		require.NoError(t, receiver.Shutdown(context.Background()))
	}
}
