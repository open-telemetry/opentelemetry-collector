// Copyright 2018, OpenCensus Authors
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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/census-instrumentation/opencensus-service/internal/compression"
	"github.com/census-instrumentation/opencensus-service/internal/factories"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := factories.GetExporterFactory(typeStr)
	require.NotNil(t, factory)

	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
}

func TestCreateMetricsExporter(t *testing.T) {
	factory := factories.GetExporterFactory(typeStr)
	cfg := factory.CreateDefaultConfig()

	_, _, err := factory.CreateMetricsExporter(cfg)
	assert.Error(t, err, factories.ErrDataTypeIsNotSupported)
}

func TestCreateTraceExporter(t *testing.T) {
	const defaultTestEndPoint = "127.0.0.1:55678"
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
				Endpoint:  defaultTestEndPoint,
				UseSecure: true,
			},
		},
		{
			name: "ReconnectionDelay",
			config: ConfigV2{
				Endpoint:          defaultTestEndPoint,
				ReconnectionDelay: 5 * time.Second,
			},
		},
		{
			name: "KeepaliveParameters",
			config: ConfigV2{
				Endpoint: defaultTestEndPoint,
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
				Endpoint:    defaultTestEndPoint,
				Compression: compression.Gzip,
			},
		},
		{
			name: "Headers",
			config: ConfigV2{
				Endpoint: defaultTestEndPoint,
				Headers: map[string]string{
					"hdr1": "val1",
					"hdr2": "val2",
				},
			},
		},
		{
			name: "NumWorkers",
			config: ConfigV2{
				Endpoint:   defaultTestEndPoint,
				NumWorkers: 3,
			},
		},
		{
			name: "CompressionError",
			config: ConfigV2{
				Endpoint:    defaultTestEndPoint,
				Compression: "unknown compression",
			},
			mustFail: true,
		},
		{
			name: "CertPemFile",
			config: ConfigV2{
				Endpoint:    defaultTestEndPoint,
				CertPemFile: "testdata/test_cert.pem",
			},
		},
		{
			name: "CertPemFileError",
			config: ConfigV2{
				Endpoint:    defaultTestEndPoint,
				CertPemFile: "nosuchfile",
			},
			mustFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := factories.GetExporterFactory(typeStr)
			consumer, stopFunc, err := factory.CreateTraceExporter(&tt.config)

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
