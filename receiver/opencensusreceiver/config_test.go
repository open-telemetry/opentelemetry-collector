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
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/config"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/receiver"
)

func TestLoadConfig(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.Nil(t, err)

	factory := &Factory{}
	factories.Receivers[typeStr] = factory
	cfg, err := config.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Currently disabled receivers are removed from the total list of receivers so 'opencensus/disabled' doesn't
	// contribute to the count.
	assert.Equal(t, len(cfg.Receivers), 6)

	r0 := cfg.Receivers["opencensus"]
	assert.Equal(t, r0, factory.CreateDefaultConfig())

	r1 := cfg.Receivers["opencensus/customname"].(*Config)
	assert.Equal(t, r1.ReceiverSettings,
		configmodels.ReceiverSettings{
			TypeVal:  typeStr,
			NameVal:  "opencensus/customname",
			Endpoint: "0.0.0.0:9090",
		})

	r2 := cfg.Receivers["opencensus/keepalive"].(*Config)
	assert.Equal(t, r2,
		&Config{
			SecureReceiverSettings: receiver.SecureReceiverSettings{
				ReceiverSettings: configmodels.ReceiverSettings{
					TypeVal:  typeStr,
					NameVal:  "opencensus/keepalive",
					Endpoint: "localhost:55678",
				},
				TLSCredentials: nil,
			},
			Keepalive: &serverParametersAndEnforcementPolicy{
				ServerParameters: &keepaliveServerParameters{
					MaxConnectionIdle:     11 * time.Second,
					MaxConnectionAge:      12 * time.Second,
					MaxConnectionAgeGrace: 13 * time.Second,
					Time:                  30 * time.Second,
					Timeout:               5 * time.Second,
				},
				EnforcementPolicy: &keepaliveEnforcementPolicy{
					MinTime:             10 * time.Second,
					PermitWithoutStream: true,
				},
			},
		})

	r3 := cfg.Receivers["opencensus/msg-size-conc-connect-max-idle"].(*Config)
	assert.Equal(t, r3,
		&Config{
			SecureReceiverSettings: receiver.SecureReceiverSettings{
				ReceiverSettings: configmodels.ReceiverSettings{
					TypeVal:  typeStr,
					NameVal:  "opencensus/msg-size-conc-connect-max-idle",
					Endpoint: "localhost:55678",
				},
			},
			MaxRecvMsgSizeMiB:    32,
			MaxConcurrentStreams: 16,
			Keepalive: &serverParametersAndEnforcementPolicy{
				ServerParameters: &keepaliveServerParameters{
					MaxConnectionIdle: 10 * time.Second,
				},
			},
		})

	// TODO(ccaraman): Once the config loader checks for the files existence, this test may fail and require
	// 	use of fake cert/key for test purposes.
	r4 := cfg.Receivers["opencensus/tlscredentials"].(*Config)
	assert.Equal(t, r4,
		&Config{
			SecureReceiverSettings: receiver.SecureReceiverSettings{
				ReceiverSettings: configmodels.ReceiverSettings{
					TypeVal:  typeStr,
					NameVal:  "opencensus/tlscredentials",
					Endpoint: "localhost:55678",
				},
				TLSCredentials: &receiver.TLSCredentials{
					CertFile: "test.crt",
					KeyFile:  "test.key",
				},
			},
		})

	r5 := cfg.Receivers["opencensus/cors"].(*Config)
	assert.Equal(t, r5,
		&Config{
			SecureReceiverSettings: receiver.SecureReceiverSettings{
				ReceiverSettings: configmodels.ReceiverSettings{
					TypeVal:  typeStr,
					NameVal:  "opencensus/cors",
					Endpoint: "localhost:55678",
				},
			},
			CorsOrigins: []string{"https://*.test.com", "https://test.com"},
		})
}
