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

package opencensusreceiver

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configcheck"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/receiver"
	"github.com/open-telemetry/opentelemetry-collector/testutils"
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

	tReceiver, err := factory.CreateTraceReceiver(context.Background(), zap.NewNop(), cfg, nil)
	assert.NotNil(t, tReceiver)
	assert.Nil(t, err)

	mReceiver, err := factory.CreateMetricsReceiver(zap.NewNop(), cfg, nil)
	assert.NotNil(t, mReceiver)
	assert.Nil(t, err)
}

func TestCreateTraceReceiver(t *testing.T) {
	factory := Factory{}
	endpoint := testutils.GetAvailableLocalAddress(t)
	defaultReceiverSettings := configmodels.ReceiverSettings{
		TypeVal:  typeStr,
		NameVal:  typeStr,
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
				SecureReceiverSettings: receiver.SecureReceiverSettings{
					ReceiverSettings: defaultReceiverSettings,
					TLSCredentials:   nil,
				},
			},
		},
		{
			name: "invalid_port",
			cfg: &Config{
				SecureReceiverSettings: receiver.SecureReceiverSettings{
					ReceiverSettings: configmodels.ReceiverSettings{
						TypeVal:  typeStr,
						NameVal:  typeStr,
						Endpoint: "localhost:112233",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "max-msg-size-and-concurrent-connections",
			cfg: &Config{
				SecureReceiverSettings: receiver.SecureReceiverSettings{
					ReceiverSettings: defaultReceiverSettings,
				},
				MaxRecvMsgSizeMiB:    32,
				MaxConcurrentStreams: 16,
			},
		},
	}
	ctx := context.Background()
	logger := zap.NewNop()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := new(exportertest.SinkTraceExporter)
			tr, err := factory.CreateTraceReceiver(ctx, logger, tt.cfg, sink)
			if (err != nil) != tt.wantErr {
				t.Errorf("factory.CreateTraceReceiver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tr != nil {
				mh := component.NewMockHost()
				err := tr.Start(mh)
				require.NoError(t, err, "Start() error = %v", err)
				tr.Shutdown()
			}
		})
	}
}

func TestCreateMetricReceiver(t *testing.T) {
	factory := Factory{}
	endpoint := testutils.GetAvailableLocalAddress(t)
	defaultReceiverSettings := configmodels.ReceiverSettings{
		TypeVal:  typeStr,
		NameVal:  typeStr,
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
				SecureReceiverSettings: receiver.SecureReceiverSettings{
					ReceiverSettings: defaultReceiverSettings,
				},
			},
		},
		{
			name: "invalid_address",
			cfg: &Config{
				SecureReceiverSettings: receiver.SecureReceiverSettings{
					ReceiverSettings: configmodels.ReceiverSettings{
						TypeVal:  typeStr,
						NameVal:  typeStr,
						Endpoint: "327.0.0.1:1122",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "keepalive",
			cfg: &Config{
				SecureReceiverSettings: receiver.SecureReceiverSettings{
					ReceiverSettings: defaultReceiverSettings,
				},
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
	logger := zap.NewNop()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := new(exportertest.SinkMetricsExporter)
			tc, err := factory.CreateMetricsReceiver(logger, tt.cfg, sink)
			if (err != nil) != tt.wantErr {
				t.Errorf("factory.CreateMetricsReceiver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tc != nil {
				mh := component.NewMockHost()
				err := tc.Start(mh)
				require.NoError(t, err, "Start() error = %v", err)
				tc.Shutdown()
			}
		})
	}
}
