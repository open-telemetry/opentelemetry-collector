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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configprotocol"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/testutils"
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
	config.Endpoint = testutils.GetAvailableLocalAddress(t)

	creationParams := component.ReceiverCreateParams{Logger: zap.NewNop()}
	tReceiver, err := factory.CreateTraceReceiver(context.Background(), creationParams, cfg, nil)
	assert.NotNil(t, tReceiver)
	assert.NoError(t, err)

	mReceiver, err := factory.CreateMetricsReceiver(context.Background(), creationParams, cfg, nil)
	assert.NotNil(t, mReceiver)
	assert.NoError(t, err)
}

func TestCreateTraceReceiver(t *testing.T) {
	factory := Factory{}
	endpoint := testutils.GetAvailableLocalAddress(t)
	defaultReceiverSettings := configmodels.ReceiverSettings{
		TypeVal: typeStr,
		NameVal: typeStr,
	}
	defaultProtocolSettings := configprotocol.ProtocolServerSettings{
		Endpoint: endpoint,
	}

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "default",
			cfg: &Config{
				ReceiverSettings:       defaultReceiverSettings,
				ProtocolServerSettings: defaultProtocolSettings,
				Transport:              "tcp",
			},
		},
		{
			name: "invalid_port",
			cfg: &Config{
				ReceiverSettings: configmodels.ReceiverSettings{
					TypeVal: typeStr,
					NameVal: typeStr,
				},
				ProtocolServerSettings: configprotocol.ProtocolServerSettings{
					Endpoint: "localhost:112233",
				},
				Transport: "tcp",
			},
			wantErr: true,
		},
		{
			name: "max-msg-size-and-concurrent-connections",
			cfg: &Config{
				ReceiverSettings:     defaultReceiverSettings,
				Transport:            "tcp",
				MaxRecvMsgSizeMiB:    32,
				MaxConcurrentStreams: 16,
			},
		},
	}
	ctx := context.Background()
	creationParams := component.ReceiverCreateParams{Logger: zap.NewNop()}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := new(exportertest.SinkTraceExporter)
			tr, err := factory.CreateTraceReceiver(ctx, creationParams, tt.cfg, sink)
			if (err != nil) != tt.wantErr {
				t.Errorf("factory.CreateTraceReceiver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tr != nil {
				require.NoError(t, tr.Start(context.Background(), componenttest.NewNopHost()), "Start() error = %v", err)
				tr.Shutdown(context.Background())
			}
		})
	}
}

func TestCreateMetricReceiver(t *testing.T) {
	factory := Factory{}
	endpoint := testutils.GetAvailableLocalAddress(t)
	defaultReceiverSettings := configmodels.ReceiverSettings{
		TypeVal: typeStr,
		NameVal: typeStr,
	}
	defaultProtocolSettings := configprotocol.ProtocolServerSettings{
		Endpoint: endpoint,
	}
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "default",
			cfg: &Config{
				ReceiverSettings:       defaultReceiverSettings,
				ProtocolServerSettings: defaultProtocolSettings,
				Transport:              "tcp",
			},
		},
		{
			name: "invalid_address",
			cfg: &Config{
				ReceiverSettings: configmodels.ReceiverSettings{
					TypeVal: typeStr,
					NameVal: typeStr,
				},
				ProtocolServerSettings: configprotocol.ProtocolServerSettings{
					Endpoint: "327.0.0.1:1122",
				},
				Transport: "tcp",
			},
			wantErr: true,
		},
		{
			name: "keepalive",
			cfg: &Config{
				ReceiverSettings: defaultReceiverSettings,
				Transport:        "tcp",
				Keepalive: &serverParametersAndEnforcementPolicy{
					ServerParameters: &keepaliveServerParameters{
						MaxConnectionAge: 60 * time.Second,
					},
					EnforcementPolicy: &keepaliveEnforcementPolicy{
						MinTime:             30 * time.Second,
						PermitWithoutStream: true,
					},
				},
			},
		},
	}
	ctx := context.Background()
	creationParams := component.ReceiverCreateParams{Logger: zap.NewNop()}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := new(exportertest.SinkMetricsExporter)
			tc, err := factory.CreateMetricsReceiver(ctx, creationParams, tt.cfg, sink)
			if (err != nil) != tt.wantErr {
				t.Errorf("factory.CreateMetricsReceiver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tc != nil {
				require.NoError(t, tc.Start(context.Background(), componenttest.NewNopHost()), "Start() error = %v", err)
				tc.Shutdown(context.Background())
			}
		})
	}
}
