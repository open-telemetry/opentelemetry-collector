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

package fileexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configcheck"
)

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateMetricsExporter(t *testing.T) {
	cfg := createDefaultConfig()
	exp, err := createMetricsExporter(
		context.Background(),
		componenttest.NewNopExporterCreateSettings(),
		cfg)
	assert.NoError(t, err)
	require.NotNil(t, exp)
}

func TestCreateTracesExporter(t *testing.T) {
	cfg := createDefaultConfig()
	exp, err := createTracesExporter(
		context.Background(),
		componenttest.NewNopExporterCreateSettings(),
		cfg)
	assert.NoError(t, err)
	require.NotNil(t, exp)
}

func TestCreateLogsExporter(t *testing.T) {
	cfg := createDefaultConfig()
	exp, err := createLogsExporter(
		context.Background(),
		componenttest.NewNopExporterCreateSettings(),
		cfg)
	assert.NoError(t, err)
	require.NotNil(t, exp)
}
