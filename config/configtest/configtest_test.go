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

package configtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
)

func TestLoadConfigFile(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	cfg, err := LoadConfigFile(t, "testdata/config.yaml", factories)
	require.NoError(t, err, "Unable to load config")

	// Verify extensions.
	require.Len(t, cfg.Extensions, 2)
	assert.Contains(t, cfg.Extensions, "nop")
	assert.Contains(t, cfg.Extensions, "nop/myextension")

	// Verify receivers
	require.Len(t, cfg.Receivers, 2)
	assert.Contains(t, cfg.Receivers, config.NewID("nop"))
	assert.Contains(t, cfg.Receivers, config.NewIDWithName("nop", "myreceiver"))

	// Verify exporters
	assert.Len(t, cfg.Exporters, 2)
	assert.Contains(t, cfg.Exporters, "nop")
	assert.Contains(t, cfg.Exporters, "nop/myexporter")

	// Verify Processors
	assert.Len(t, cfg.Processors, 2)
	assert.Contains(t, cfg.Processors, "nop")
	assert.Contains(t, cfg.Processors, "nop/myprocessor")

	// Verify service.
	require.Len(t, cfg.Service.Extensions, 1)
	assert.Contains(t, cfg.Service.Extensions, "nop")
	require.Len(t, cfg.Service.Pipelines, 1)
	assert.Equal(t,
		&config.Pipeline{
			Name:       "traces",
			InputType:  config.TracesDataType,
			Receivers:  []config.ComponentID{config.NewID("nop")},
			Processors: []string{"nop"},
			Exporters:  []string{"nop"},
		},
		cfg.Service.Pipelines["traces"],
		"Did not load pipeline config correctly")
}
