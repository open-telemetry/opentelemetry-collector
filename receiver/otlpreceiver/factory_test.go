// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))
}

func TestCreateSameReceiver(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPC.NetAddr.Endpoint = testutil.GetAvailableLocalAddress(t)
	cfg.HTTP.Endpoint = testutil.GetAvailableLocalAddress(t)

	creationSet := receivertest.NewNopCreateSettings()
	tReceiver, err := factory.CreateTracesReceiver(context.Background(), creationSet, cfg, consumertest.NewNop())
	assert.NotNil(t, tReceiver)
	assert.NoError(t, err)

	mReceiver, err := factory.CreateMetricsReceiver(context.Background(), creationSet, cfg, consumertest.NewNop())
	assert.NotNil(t, mReceiver)
	assert.NoError(t, err)

	lReceiver, err := factory.CreateMetricsReceiver(context.Background(), creationSet, cfg, consumertest.NewNop())
	assert.NotNil(t, lReceiver)
	assert.NoError(t, err)

	assert.Same(t, tReceiver, mReceiver)
	assert.Same(t, tReceiver, lReceiver)
}

func TestCreateTracesReceiver(t *testing.T) {
	factory := NewFactory()
	defaultGRPCSettings := &configgrpc.ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  testutil.GetAvailableLocalAddress(t),
			Transport: "tcp",
		},
	}
	defaultHTTPSettings := &HTTPConfig{
		ServerConfig: &confighttp.ServerConfig{
			Endpoint: testutil.GetAvailableLocalAddress(t),
		},
		TracesURLPath:  defaultTracesURLPath,
		MetricsURLPath: defaultMetricsURLPath,
		LogsURLPath:    defaultLogsURLPath,
	}

	tests := []struct {
		name         string
		cfg          *Config
		wantStartErr bool
		wantErr      bool
		sink         consumer.Traces
	}{
		{
			name: "default",
			cfg: &Config{
				Protocols: Protocols{
					GRPC: defaultGRPCSettings,
					HTTP: defaultHTTPSettings,
				},
			},
			sink: consumertest.NewNop(),
		},
		{
			name: "invalid_grpc_port",
			cfg: &Config{
				Protocols: Protocols{
					GRPC: &configgrpc.ServerConfig{
						NetAddr: confignet.AddrConfig{
							Endpoint:  "localhost:112233",
							Transport: "tcp",
						},
					},
					HTTP: defaultHTTPSettings,
				},
			},
			wantStartErr: true,
			sink:         consumertest.NewNop(),
		},
		{
			name: "invalid_http_port",
			cfg: &Config{
				Protocols: Protocols{
					GRPC: defaultGRPCSettings,
					HTTP: &HTTPConfig{
						ServerConfig: &confighttp.ServerConfig{
							Endpoint: "localhost:112233",
						},
						TracesURLPath: defaultTracesURLPath,
					},
				},
			},
			wantStartErr: true,
			sink:         consumertest.NewNop(),
		},
		{
			name: "no_http_or_grcp_config",
			cfg: &Config{
				Protocols: Protocols{},
			},
			sink: consumertest.NewNop(),
		},
	}
	ctx := context.Background()
	creationSet := receivertest.NewNopCreateSettings()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, err := factory.CreateTracesReceiver(ctx, creationSet, tt.cfg, tt.sink)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.wantStartErr {
				assert.Error(t, tr.Start(context.Background(), componenttest.NewNopHost()))
			} else {
				assert.NoError(t, tr.Start(context.Background(), componenttest.NewNopHost()))
				assert.NoError(t, tr.Shutdown(context.Background()))
			}
		})
	}
}

func TestCreateMetricReceiver(t *testing.T) {
	factory := NewFactory()
	defaultGRPCSettings := &configgrpc.ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  testutil.GetAvailableLocalAddress(t),
			Transport: "tcp",
		},
	}
	defaultHTTPSettings := &HTTPConfig{
		ServerConfig: &confighttp.ServerConfig{
			Endpoint: testutil.GetAvailableLocalAddress(t),
		},
		TracesURLPath:  defaultTracesURLPath,
		MetricsURLPath: defaultMetricsURLPath,
		LogsURLPath:    defaultLogsURLPath,
	}

	tests := []struct {
		name         string
		cfg          *Config
		wantStartErr bool
		wantErr      bool
		sink         consumer.Metrics
	}{
		{
			name: "default",
			cfg: &Config{
				Protocols: Protocols{
					GRPC: defaultGRPCSettings,
					HTTP: defaultHTTPSettings,
				},
			},
			sink: consumertest.NewNop(),
		},
		{
			name: "invalid_grpc_address",
			cfg: &Config{
				Protocols: Protocols{
					GRPC: &configgrpc.ServerConfig{
						NetAddr: confignet.AddrConfig{
							Endpoint:  "327.0.0.1:1122",
							Transport: "tcp",
						},
					},
					HTTP: defaultHTTPSettings,
				},
			},
			wantStartErr: true,
			sink:         consumertest.NewNop(),
		},
		{
			name: "invalid_http_address",
			cfg: &Config{
				Protocols: Protocols{
					GRPC: defaultGRPCSettings,
					HTTP: &HTTPConfig{
						ServerConfig: &confighttp.ServerConfig{
							Endpoint: "327.0.0.1:1122",
						},
						MetricsURLPath: defaultMetricsURLPath,
					},
				},
			},
			wantStartErr: true,
			sink:         consumertest.NewNop(),
		},
		{
			name: "no_http_or_grcp_config",
			cfg: &Config{
				Protocols: Protocols{},
			},
			sink: consumertest.NewNop(),
		},
	}
	ctx := context.Background()
	creationSet := receivertest.NewNopCreateSettings()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr, err := factory.CreateMetricsReceiver(ctx, creationSet, tt.cfg, tt.sink)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.wantStartErr {
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
	defaultGRPCSettings := &configgrpc.ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  testutil.GetAvailableLocalAddress(t),
			Transport: "tcp",
		},
	}
	defaultHTTPSettings := &HTTPConfig{
		ServerConfig: &confighttp.ServerConfig{
			Endpoint: testutil.GetAvailableLocalAddress(t),
		},
		TracesURLPath:  defaultTracesURLPath,
		MetricsURLPath: defaultMetricsURLPath,
		LogsURLPath:    defaultLogsURLPath,
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
				Protocols: Protocols{
					GRPC: defaultGRPCSettings,
					HTTP: defaultHTTPSettings,
				},
			},
			sink: consumertest.NewNop(),
		},
		{
			name: "invalid_grpc_address",
			cfg: &Config{
				Protocols: Protocols{
					GRPC: &configgrpc.ServerConfig{
						NetAddr: confignet.AddrConfig{
							Endpoint:  "327.0.0.1:1122",
							Transport: "tcp",
						},
					},
					HTTP: defaultHTTPSettings,
				},
			},
			wantStartErr: true,
			sink:         consumertest.NewNop(),
		},
		{
			name: "invalid_http_address",
			cfg: &Config{
				Protocols: Protocols{
					GRPC: defaultGRPCSettings,
					HTTP: &HTTPConfig{
						ServerConfig: &confighttp.ServerConfig{
							Endpoint: "327.0.0.1:1122",
						},
						LogsURLPath: defaultLogsURLPath,
					},
				},
			},
			wantStartErr: true,
			sink:         consumertest.NewNop(),
		},
		{
			name: "no_http_or_grcp_config",
			cfg: &Config{
				Protocols: Protocols{},
			},
			sink: consumertest.NewNop(),
		},
	}
	ctx := context.Background()
	creationSet := receivertest.NewNopCreateSettings()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr, err := factory.CreateLogsReceiver(ctx, creationSet, tt.cfg, tt.sink)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.wantStartErr {
				assert.Error(t, mr.Start(context.Background(), componenttest.NewNopHost()))
			} else {
				require.NoError(t, mr.Start(context.Background(), componenttest.NewNopHost()))
				assert.NoError(t, mr.Shutdown(context.Background()))
			}
		})
	}
}
