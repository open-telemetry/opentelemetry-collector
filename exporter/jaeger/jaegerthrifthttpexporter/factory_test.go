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

package jaegerthrifthttpexporter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
}

func TestCreateMetricsExporter(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()

	_, err := factory.CreateMetricsExporter(zap.NewNop(), cfg)
	assert.Error(t, err, configerror.ErrDataTypeIsNotSupported)
}

func TestCreateInstanceViaFactory(t *testing.T) {
	factory := Factory{}

	cfg := factory.CreateDefaultConfig()

	// Default config doesn't have default URL so creating from it should
	// fail.
	exp, err := factory.CreateTraceExporter(
		zap.NewNop(),
		cfg)
	assert.Error(t, err)
	assert.Nil(t, exp)

	// Endpoint doesn't have a default value so set it directly.
	expCfg := cfg.(*Config)
	expCfg.URL = "http://some.target.org:12345/api/traces"
	exp, err = factory.CreateTraceExporter(
		zap.NewNop(),
		cfg)
	assert.NoError(t, err)
	assert.NotNil(t, exp)

	assert.NoError(t, exp.Shutdown())
}

func TestFactory_CreateTraceExporter(t *testing.T) {
	f := &Factory{}
	config := &Config{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		URL: "http://some.other.location/api/traces",
		Headers: map[string]string{
			"added-entry": "added value",
			"dot.test":    "test",
		},
		Timeout: 2 * time.Second,
	}

	te, err := f.CreateTraceExporter(zap.NewNop(), config)
	assert.NoError(t, err)
	assert.NotNil(t, te)
}

func TestFactory_CreateTraceExporterFails(t *testing.T) {
	tests := []struct {
		name         string
		config       *Config
		errorMessage string
	}{
		{
			name: "empty_url",
			config: &Config{
				ExporterSettings: configmodels.ExporterSettings{
					TypeVal: typeStr,
					NameVal: typeStr,
				},
			},
			errorMessage: "\"jaeger_thrift_http\" config requires a valid \"url\": parse \"\": empty url",
		},
		{
			name: "invalid_url",
			config: &Config{
				ExporterSettings: configmodels.ExporterSettings{
					TypeVal: typeStr,
					NameVal: typeStr,
				},
				URL: ".localhost:123",
			},
			errorMessage: "\"jaeger_thrift_http\" config requires a valid \"url\": parse \".localhost:123\": invalid URI for request",
		},
		{
			name: "negative_duration",
			config: &Config{
				ExporterSettings: configmodels.ExporterSettings{
					TypeVal: typeStr,
					NameVal: typeStr,
				},
				URL:     "localhost:123",
				Timeout: -2 * time.Second,
			},
			errorMessage: "\"jaeger_thrift_http\" config requires a positive value for \"timeout\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Factory{}
			te, err := f.CreateTraceExporter(zap.NewNop(), tt.config)
			assert.EqualError(t, err, tt.errorMessage)
			assert.Nil(t, te)
		})
	}
}
