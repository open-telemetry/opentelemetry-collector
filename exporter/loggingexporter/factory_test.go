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

package loggingexporter

import (
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-service/factories"
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

	_, _, err := factory.CreateMetricsExporter(zap.NewNop(), cfg)
	assert.Nil(t, err)
}

func TestCreateTraceExporter(t *testing.T) {
	factory := factories.GetExporterFactory(typeStr)
	cfg := factory.CreateDefaultConfig()

	_, _, err := factory.CreateTraceExporter(zap.NewNop(), cfg)
	assert.Nil(t, err)
}
