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

package otlpreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testutil"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateReceiver(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPC.NetAddr.Endpoint = testutil.GetAvailableLocalAddress(t)
	cfg.HTTP.Endpoint = testutil.GetAvailableLocalAddress(t)

	creationSet := componenttest.NewNopReceiverCreateSettings()
	tReceiver, err := factory.CreateTracesReceiver(context.Background(), creationSet, cfg, consumertest.NewNop())
	assert.NotNil(t, tReceiver)
	assert.NoError(t, err)

	mReceiver, err := factory.CreateMetricsReceiver(context.Background(), creationSet, cfg, consumertest.NewNop())
	assert.NotNil(t, mReceiver)
	assert.NoError(t, err)
}

func TestCreateTracesReceiver(t *testing.T) {
	factory := NewFactory()
	defaultGRPCSettings := &configgrpc.GRPCServerSettings{
		NetAddr: confignet.NetAddr{
			Endpoint:  testutil.GetAvailableLocalAddress(t),
			Transport: "tcp",
		},
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
				ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
				Protocols: Protocols{
					GRPC: defaultGRPCSettings,
					HTTP: defaultHTTPSettings,
				},
			},
		},
		{
			name: "invalid_grpc_port",
			cfg: &Config{
				ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
				Protocols: Protocols{
					GRPC: &configgrpc.GRPCServerSettings{
						NetAddr: confignet.NetAddr{
							Endpoint:  "localhost:112233",
							Transport: "tcp",
						},
					},
					HTTP: defaultHTTPSettings,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid_http_port",
			cfg: &Config{
				ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
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
	creationSet := componenttest.NewNopReceiverCreateSettings()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := new(consumertest.TracesSink)
			tr, err := factory.CreateTracesReceiver(ctx, creationSet, tt.cfg, sink)
			assert.NoError(t, err)
			require.NotNil(t, tr)
			if tt.wantErr {
				assert.Error(t, tr.Start(context.Background(), componenttest.NewNopHost()))
				assert.NoError(t, tr.Shutdown(context.Background()))
			} else {
				assert.NoError(t, tr.Start(context.Background(), componenttest.NewNopHost()))
				assert.NoError(t, tr.Shutdown(context.Background()))
			}
		})
	}
}

func TestCreateMetricReceiver(t *testing.T) {
	factory := NewFactory()
	defaultGRPCSettings := &configgrpc.GRPCServerSettings{
		NetAddr: confignet.NetAddr{
			Endpoint:  testutil.GetAvailableLocalAddress(t),
			Transport: "tcp",
		},
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
				ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
				Protocols: Protocols{
					GRPC: defaultGRPCSettings,
					HTTP: defaultHTTPSettings,
				},
			},
		},
		{
			name: "invalid_grpc_address",
			cfg: &Config{
				ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
				Protocols: Protocols{
					GRPC: &configgrpc.GRPCServerSettings{
						NetAddr: confignet.NetAddr{
							Endpoint:  "327.0.0.1:1122",
							Transport: "tcp",
						},
					},
					HTTP: defaultHTTPSettings,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid_http_address",
			cfg: &Config{
				ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
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
	creationSet := componenttest.NewNopReceiverCreateSettings()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := new(consumertest.MetricsSink)
			mr, err := factory.CreateMetricsReceiver(ctx, creationSet, tt.cfg, sink)
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

func TestCreateLogReceiver(t *testing.T) {
	factory := NewFactory()
	defaultGRPCSettings := &configgrpc.GRPCServerSettings{
		NetAddr: confignet.NetAddr{
			Endpoint:  testutil.GetAvailableLocalAddress(t),
			Transport: "tcp",
		},
	}
	defaultHTTPSettings := &confighttp.HTTPServerSettings{
		Endpoint: testutil.GetAvailableLocalAddress(t),
	}

	tests := []struct {
		name         string
		cfg          *Config
		wantStartErr bool
		wantErr      bool
		sink         consumer.Logs
	}{
		{
			name: "default",
			cfg: &Config{
				ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
				Protocols: Protocols{
					GRPC: defaultGRPCSettings,
					HTTP: defaultHTTPSettings,
				},
			},
			sink: new(consumertest.LogsSink),
		},
		{
			name: "invalid_grpc_address",
			cfg: &Config{
				ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
				Protocols: Protocols{
					GRPC: &configgrpc.GRPCServerSettings{
						NetAddr: confignet.NetAddr{
							Endpoint:  "327.0.0.1:1122",
							Transport: "tcp",
						},
					},
					HTTP: defaultHTTPSettings,
				},
			},
			wantStartErr: true,
			sink:         new(consumertest.LogsSink),
		},
		{
			name: "invalid_http_address",
			cfg: &Config{
				ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
				Protocols: Protocols{
					GRPC: defaultGRPCSettings,
					HTTP: &confighttp.HTTPServerSettings{
						Endpoint: "327.0.0.1:1122",
					},
				},
			},
			wantStartErr: true,
			sink:         new(consumertest.LogsSink),
		},
		{
			name: "no_next_consumer",
			cfg: &Config{
				ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
				Protocols: Protocols{
					GRPC: defaultGRPCSettings,
					HTTP: &confighttp.HTTPServerSettings{
						Endpoint: "327.0.0.1:1122",
					},
				},
			},
			wantErr: true,
			sink:    nil,
		},
		{
			name: "no_http_or_grcp_config",
			cfg: &Config{
				ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
				Protocols:        Protocols{},
			},
			wantErr: false,
			sink:    new(consumertest.LogsSink),
		},
	}
	ctx := context.Background()
	creationSet := componenttest.NewNopReceiverCreateSettings()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr, err := factory.CreateLogsReceiver(ctx, creationSet, tt.cfg, tt.sink)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			require.NotNil(t, mr)

			if tt.wantStartErr {
				assert.Error(t, mr.Start(context.Background(), componenttest.NewNopHost()))
				assert.NoError(t, mr.Shutdown(context.Background()))
			} else {
				require.NoError(t, mr.Start(context.Background(), componenttest.NewNopHost()))
				assert.NoError(t, mr.Shutdown(context.Background()))
			}
		})
	}
}
