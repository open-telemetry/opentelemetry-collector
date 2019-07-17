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

package prometheusexporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/config/configerror"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := Factory{}
	require.NotNil(t, factory)

	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
}

func TestCreateTraceExporter(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()

	_, _, err := factory.CreateTraceExporter(zap.NewNop(), cfg)
	assert.Error(t, err, configerror.ErrDataTypeIsNotSupported)
}

func TestCreateMetricsExporter(t *testing.T) {
	const defaultTestEndPoint = "127.0.0.1:55678"
	tests := []struct {
		name     string
		config   Config
		mustFail bool
	}{
		{
			name: "NoEndpoint",
			config: Config{
				Endpoint: "",
			},
			mustFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := Factory{}
			consumer, stopFunc, err := factory.CreateMetricsExporter(zap.NewNop(), &tt.config)

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
