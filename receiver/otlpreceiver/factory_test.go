// Copyright The OpenTelemetry Authors
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

package otlpreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/testutil"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateReceiver(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()

	config := cfg.(*Config)
	config.GRPC.Endpoint = testutil.GetAvailableLocalAddress(t)
	config.HTTP.Endpoint = testutil.GetAvailableLocalAddress(t)

	creationParams := component.ReceiverCreateParams{Logger: zap.NewNop()}
	tReceiver, err := factory.CreateTraceReceiver(context.Background(), creationParams, cfg, new(exportertest.SinkTraceExporter))
	assert.NotNil(t, tReceiver)
	assert.NoError(t, err)

	mReceiver, err := factory.CreateMetricsReceiver(context.Background(), creationParams, cfg, new(exportertest.SinkMetricsExporter))
	assert.NotNil(t, mReceiver)
	assert.NoError(t, err)
}

func TestCreateTraceReceiver(t *testing.T) {
	factory := Factory{}
	defaultReceiverSettings := configmodels.ReceiverSettings{
		TypeVal: typeStr,
		NameVal: typeStr,
	}
	defaultGRPCSettings := &configgrpc.GRPCServerSettings{
		Endpoint: testutil.GetAvailableLocalAddress(t),
	}
	defaultHTTPSettings := &confighttp.HTTPServerSettings{
		Endpoint: testutil.GetAvailableLocalAddress(t),
	}

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "default",
			cfg: &Config{
				ReceiverSettings: defaultReceiverSettings,
				Protocols: Protocols{
					GRPC: defaultGRPCSettings,
					HTTP: defaultHTTPSettings,
				},
			},
		},
		{
			name: "invalid_grpc_port",
			cfg: &Config{
				ReceiverSettings: configmodels.ReceiverSettings{
					TypeVal: typeStr,
					NameVal: typeStr,
				},
				Protocols: Protocols{
					GRPC: &configgrpc.GRPCServerSettings{
						Endpoint: "localhost:112233",
					},
					HTTP: defaultHTTPSettings,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid_http_port",
			cfg: &Config{
				ReceiverSettings: configmodels.ReceiverSettings{
					TypeVal: typeStr,
					NameVal: typeStr,
				},
				Protocols: Protocols{
					GRPC: defaultGRPCSettings,
					HTTP: &confighttp.HTTPServerSettings{
						Endpoint: "localhost:112233",
					},
				},
			},
			wantErr: true,
		},
	}
	ctx := context.Background()
	creationParams := component.ReceiverCreateParams{Logger: zap.NewNop()}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := new(exportertest.SinkTraceExporter)
			tr, err := factory.CreateTraceReceiver(ctx, creationParams, tt.cfg, sink)
			assert.NoError(t, err)
			require.NotNil(t, tr)
			if tt.wantErr {
				assert.Error(t, tr.Start(context.Background(), componenttest.NewNopHost()))
			} else {
				assert.NoError(t, tr.Start(context.Background(), componenttest.NewNopHost()))
				assert.NoError(t, tr.Shutdown(context.Background()))
			}
		})
	}
}

func TestCreateMetricReceiver(t *testing.T) {
	factory := Factory{}
	defaultReceiverSettings := configmodels.ReceiverSettings{
		TypeVal: typeStr,
		NameVal: typeStr,
	}
	defaultGRPCSettings := &configgrpc.GRPCServerSettings{
		Endpoint: testutil.GetAvailableLocalAddress(t),
	}
	defaultHTTPSettings := &confighttp.HTTPServerSettings{
		Endpoint: testutil.GetAvailableLocalAddress(t),
	}

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "default",
			cfg: &Config{
				ReceiverSettings: defaultReceiverSettings,
				Protocols: Protocols{
					GRPC: defaultGRPCSettings,
					HTTP: defaultHTTPSettings,
				},
			},
		},
		{
			name: "invalid_grpc_address",
			cfg: &Config{
				ReceiverSettings: configmodels.ReceiverSettings{
					TypeVal: typeStr,
					NameVal: typeStr,
				},
				Protocols: Protocols{
					GRPC: &configgrpc.GRPCServerSettings{
						Endpoint: "327.0.0.1:1122",
					},
					HTTP: defaultHTTPSettings,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid_http_address",
			cfg: &Config{
				ReceiverSettings: configmodels.ReceiverSettings{
					TypeVal: typeStr,
					NameVal: typeStr,
				},
				Protocols: Protocols{
					GRPC: defaultGRPCSettings,
					HTTP: &confighttp.HTTPServerSettings{
						Endpoint: "327.0.0.1:1122",
					},
				},
			},
			wantErr: true,
		},
	}
	ctx := context.Background()
	creationParams := component.ReceiverCreateParams{Logger: zap.NewNop()}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := new(exportertest.SinkMetricsExporter)
			mr, err := factory.CreateMetricsReceiver(ctx, creationParams, tt.cfg, sink)
			assert.NoError(t, err)
			require.NotNil(t, mr)
			if tt.wantErr {
				assert.Error(t, mr.Start(context.Background(), componenttest.NewNopHost()))
			} else {
				require.NoError(t, mr.Start(context.Background(), componenttest.NewNopHost()))
				assert.NoError(t, mr.Shutdown(context.Background()))
			}
		})
	}
}
