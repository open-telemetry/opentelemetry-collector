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

package jaegerexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/configv2/configerror"
	"github.com/open-telemetry/opentelemetry-service/exporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-service/internal/testutils"
	"github.com/open-telemetry/opentelemetry-service/receiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/jaegerreceiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/receivertest"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := exporter.GetFactory(typeStr)
	require.NotNil(t, factory)

	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
}

func TestCreateMetricsExporter(t *testing.T) {
	factory := exporter.GetFactory(typeStr)
	cfg := factory.CreateDefaultConfig()

	_, _, err := factory.CreateMetricsExporter(zap.NewNop(), cfg)
	assert.Error(t, err, configerror.ErrDataTypeIsNotSupported)
}

func TestCreateTraceExporter(t *testing.T) {
	// This test is about creating the exporter and stopping it. However, the
	// exporter keeps trying to update its connection state in the background
	// so unless there is a receiver enabled the stop call can return different
	// results. Standing up a receiver to ensure that stop don't report errors.
	rcvFactory := receiver.GetFactory(typeStr)
	require.NotNil(t, rcvFactory)
	rcvCfg := rcvFactory.CreateDefaultConfig().(*jaegerreceiver.ConfigV2)
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
			name: "NoCollectorEndpoint",
			config: ConfigV2{
				CollectorEndpoint: "",
			},
			mustFail: true,
		},
		{
			name: "NoUsername",
			config: ConfigV2{
				Username: "",
			},
			mustFail: true,
		},
		{
			name: "NoPassword",
			config: ConfigV2{
				Password: "",
			},
			mustFail: true,
		},
		{
			name: "NoServiceName",
			config: ConfigV2{
				ServiceName: "",
			},
			mustFail: true,
		},
		{
			name: "NumWorkers",
			config: ConfigV2{
				CollectorEndpoint: rcvCfg.CollectorEndpoint,
				NumWorkers:        3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := exporter.GetFactory(typeStr)
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
