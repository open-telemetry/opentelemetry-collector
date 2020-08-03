// Copyright 2020 The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cortexexporter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configerror"
	"go.opentelemetry.io/collector/config/configtls"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateMetricsExporter(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	params := component.ExporterCreateParams{Logger: zap.NewNop()}
	_, err := factory.CreateMetricsExporter(context.Background(), params, cfg)
	assert.Error(t, err, configerror.ErrDataTypeIsNotSupported)

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
			},
		},
		{
			name: "Compression",
			config: Config{
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint:    endpoint,
					Compression: configgrpc.CompressionGzip,
				},
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
			},
		},
		{
			name: "NumConsumers",
			config: Config{
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: endpoint,
				},
			},
		},
		{
			name: "CompressionError",
			config: Config{
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint:    endpoint,
					Compression: "unknown compression",
				},
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
			},
			mustFail: true,
		},
	}
}
