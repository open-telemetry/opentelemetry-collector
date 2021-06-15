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

package opencensusreceiver

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
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/testutil"
)

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateReceiver(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.NetAddr.Endpoint = testutil.GetAvailableLocalAddress(t)

	set := componenttest.NewNopReceiverCreateSettings()
	tReceiver, err := createTracesReceiver(context.Background(), set, cfg, nil)
	assert.NotNil(t, tReceiver)
	assert.NoError(t, err)

	mReceiver, err := createMetricsReceiver(context.Background(), set, cfg, nil)
	assert.NotNil(t, mReceiver)
	assert.NoError(t, err)
}

func TestCreateTracesReceiver(t *testing.T) {
	defaultNetAddr := confignet.NetAddr{
		Endpoint:  testutil.GetAvailableLocalAddress(t),
		Transport: "tcp",
	}
	defaultGRPCSettings := configgrpc.GRPCServerSettings{
		NetAddr: defaultNetAddr,
	}
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "default",
			cfg: &Config{
				ReceiverSettings:   config.NewReceiverSettings(config.NewID(typeStr)),
				GRPCServerSettings: defaultGRPCSettings,
			},
		},
		{
			name: "invalid_port",
			cfg: &Config{
				ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
				GRPCServerSettings: configgrpc.GRPCServerSettings{
					NetAddr: confignet.NetAddr{
						Endpoint:  "localhost:112233",
						Transport: "tcp",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "max-msg-size-and-concurrent-connections",
			cfg: &Config{
				ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
				GRPCServerSettings: configgrpc.GRPCServerSettings{
					NetAddr:              defaultNetAddr,
					MaxRecvMsgSizeMiB:    32,
					MaxConcurrentStreams: 16,
				},
			},
		},
	}
	ctx := context.Background()
	set := componenttest.NewNopReceiverCreateSettings()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, err := createTracesReceiver(ctx, set, tt.cfg, consumertest.NewNop())
			if (err != nil) != tt.wantErr {
				t.Errorf("factory.CreateTracesReceiver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tr != nil {
				require.NoError(t, tr.Start(context.Background(), componenttest.NewNopHost()))
				require.NoError(t, tr.Shutdown(context.Background()))
			}
		})
	}
}

func TestCreateMetricReceiver(t *testing.T) {
	defaultNetAddr := confignet.NetAddr{
		Endpoint:  testutil.GetAvailableLocalAddress(t),
		Transport: "tcp",
	}
	defaultGRPCSettings := configgrpc.GRPCServerSettings{
		NetAddr: defaultNetAddr,
	}

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "default",
			cfg: &Config{
				ReceiverSettings:   config.NewReceiverSettings(config.NewID(typeStr)),
				GRPCServerSettings: defaultGRPCSettings,
			},
		},
		{
			name: "invalid_address",
			cfg: &Config{
				ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
				GRPCServerSettings: configgrpc.GRPCServerSettings{
					NetAddr: confignet.NetAddr{
						Endpoint:  "327.0.0.1:1122",
						Transport: "tcp",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "keepalive",
			cfg: &Config{
				ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
				GRPCServerSettings: configgrpc.GRPCServerSettings{
					NetAddr: defaultNetAddr,
					Keepalive: &configgrpc.KeepaliveServerConfig{
						ServerParameters: &configgrpc.KeepaliveServerParameters{
							MaxConnectionAge: 60 * time.Second,
						},
						EnforcementPolicy: &configgrpc.KeepaliveEnforcementPolicy{
							MinTime:             30 * time.Second,
							PermitWithoutStream: true,
						},
					},
				},
			},
		},
	}
	set := componenttest.NewNopReceiverCreateSettings()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc, err := createMetricsReceiver(context.Background(), set, tt.cfg, consumertest.NewNop())
			if (err != nil) != tt.wantErr {
				t.Errorf("factory.CreateMetricsReceiver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tc != nil {
				require.NoError(t, tc.Start(context.Background(), componenttest.NewNopHost()))
				require.NoError(t, tc.Shutdown(context.Background()))
			}
		})
	}
}
