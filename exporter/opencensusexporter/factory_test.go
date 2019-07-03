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

package opencensusexporter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/exporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-service/internal/compression"
	"github.com/open-telemetry/opentelemetry-service/internal/testutils"
	"github.com/open-telemetry/opentelemetry-service/models"
	"github.com/open-telemetry/opentelemetry-service/receiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/opencensusreceiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/receivertest"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := exporter.GetExporterFactory(typeStr)
	require.NotNil(t, factory)

	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
}

func TestCreateMetricsExporter(t *testing.T) {
	factory := exporter.GetExporterFactory(typeStr)
	cfg := factory.CreateDefaultConfig()

	_, _, err := factory.CreateMetricsExporter(zap.NewNop(), cfg)
	assert.Error(t, err, models.ErrDataTypeIsNotSupported)
}

func TestCreateTraceExporter(t *testing.T) {
	// This test is about creating the exporter and stopping it. However, the
	// exporter keeps trying to update its connection state in the background
	// so unless there is a receiver enabled the stop call can return different
	// results. Standing up a receiver to ensure that stop don't report errors.
	rcvFactory := receiver.GetReceiverFactory(typeStr)
	require.NotNil(t, rcvFactory)
	rcvCfg := rcvFactory.CreateDefaultConfig().(*opencensusreceiver.ConfigV2)
	rcvCfg.Endpoint = testutils.GetAvailableLocalAddress(t)

	rcv, err := rcvFactory.CreateTraceReceiver(
		context.Background(),
		zap.NewNop(),
		rcvCfg,
		new(exportertest.SinkTraceExporter))
	require.NotNil(t, rcv)
	require.Nil(t, err)
	require.Nil(t, rcv.StartTraceReception(receivertest.NewMockHost()))
	defer rcv.StopTraceReception()

	tests := []struct {
		name     string
		config   ConfigV2
		mustFail bool
	}{
		{
			name: "NoEndpoint",
			config: ConfigV2{
				Endpoint: "",
			},
			mustFail: true,
		},
		{
			name: "UseSecure",
			config: ConfigV2{
				Endpoint:  rcvCfg.Endpoint,
				UseSecure: true,
			},
		},
		{
			name: "ReconnectionDelay",
			config: ConfigV2{
				Endpoint:          rcvCfg.Endpoint,
				ReconnectionDelay: 5 * time.Second,
			},
		},
		{
			name: "KeepaliveParameters",
			config: ConfigV2{
				Endpoint: rcvCfg.Endpoint,
				KeepaliveParameters: &keepaliveConfig{
					Time:                30 * time.Second,
					Timeout:             25 * time.Second,
					PermitWithoutStream: true,
				},
			},
		},
		{
			name: "Compression",
			config: ConfigV2{
				Endpoint:    rcvCfg.Endpoint,
				Compression: compression.Gzip,
			},
		},
		{
			name: "Headers",
			config: ConfigV2{
				Endpoint: rcvCfg.Endpoint,
				Headers: map[string]string{
					"hdr1": "val1",
					"hdr2": "val2",
				},
			},
		},
		{
			name: "NumWorkers",
			config: ConfigV2{
				Endpoint:   rcvCfg.Endpoint,
				NumWorkers: 3,
			},
		},
		{
			name: "CompressionError",
			config: ConfigV2{
				Endpoint:    rcvCfg.Endpoint,
				Compression: "unknown compression",
			},
			mustFail: true,
		},
		{
			name: "CertPemFile",
			config: ConfigV2{
				Endpoint:    rcvCfg.Endpoint,
				CertPemFile: "testdata/test_cert.pem",
			},
		},
		{
			name: "CertPemFileError",
			config: ConfigV2{
				Endpoint:    rcvCfg.Endpoint,
				CertPemFile: "nosuchfile",
			},
			mustFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := exporter.GetExporterFactory(typeStr)
			consumer, stopFunc, err := factory.CreateTraceExporter(zap.NewNop(), &tt.config)

			if tt.mustFail {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, consumer)
				assert.NotNil(t, stopFunc)

				err = stopFunc()
				assert.Nil(t, err)
			}
		})
	}
}
