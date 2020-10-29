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
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configmodels"
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
	cfg := createDefaultConfig()

	config := cfg.(*Config)
	config.NetAddr.Endpoint = testutil.GetAvailableLocalAddress(t)

	params := component.ReceiverCreateParams{Logger: zap.NewNop()}
	tReceiver, err := createTraceReceiver(context.Background(), params, cfg, nil)
	assert.NotNil(t, tReceiver)
	assert.NoError(t, err)

	mReceiver, err := createMetricsReceiver(context.Background(), params, cfg, nil)
	assert.NotNil(t, mReceiver)
	assert.NoError(t, err)
}

func TestCreateTraceReceiver(t *testing.T) {
	defaultReceiverSettings := configmodels.ReceiverSettings{
		TypeVal: typeStr,
		NameVal: typeStr,
	}
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
				ReceiverSettings:   defaultReceiverSettings,
				GRPCServerSettings: defaultGRPCSettings,
			},
		},
		{
			name: "invalid_port",
			cfg: &Config{
				ReceiverSettings: configmodels.ReceiverSettings{
					TypeVal: typeStr,
					NameVal: typeStr,
				},
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
				ReceiverSettings: defaultReceiverSettings,
				GRPCServerSettings: configgrpc.GRPCServerSettings{
					NetAddr:              defaultNetAddr,
					MaxRecvMsgSizeMiB:    32,
					MaxConcurrentStreams: 16,
				},
			},
		},
	}
	ctx := context.Background()
	params := component.ReceiverCreateParams{Logger: zap.NewNop()}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, err := createTraceReceiver(ctx, params, tt.cfg, consumertest.NewTracesNop())
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
	defaultReceiverSettings := configmodels.ReceiverSettings{
		TypeVal: typeStr,
		NameVal: typeStr,
	}
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
				ReceiverSettings:   defaultReceiverSettings,
				GRPCServerSettings: defaultGRPCSettings,
			},
		},
		{
			name: "invalid_address",
			cfg: &Config{
				ReceiverSettings: configmodels.ReceiverSettings{
					TypeVal: typeStr,
					NameVal: typeStr,
				},
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
				ReceiverSettings: defaultReceiverSettings,
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
	params := component.ReceiverCreateParams{Logger: zap.NewNop()}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc, err := createMetricsReceiver(context.Background(), params, tt.cfg, consumertest.NewMetricsNop())
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
