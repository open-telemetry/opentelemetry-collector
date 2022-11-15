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

package servicetest

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/component/id"
	"go.opentelemetry.io/collector/config"
)

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	cfg, err := LoadConfig(filepath.Join("testdata", "config.yaml"), factories)
	require.NoError(t, err)

	// Verify extensions.
	require.Len(t, cfg.Extensions, 2)
	assert.Contains(t, cfg.Extensions, id.NewID("nop"))
	assert.Contains(t, cfg.Extensions, id.NewIDWithName("nop", "myextension"))

	// Verify receivers
	require.Len(t, cfg.Receivers, 2)
	assert.Contains(t, cfg.Receivers, id.NewID("nop"))
	assert.Contains(t, cfg.Receivers, id.NewIDWithName("nop", "myreceiver"))

	// Verify exporters
	assert.Len(t, cfg.Exporters, 2)
	assert.Contains(t, cfg.Exporters, id.NewID("nop"))
	assert.Contains(t, cfg.Exporters, id.NewIDWithName("nop", "myexporter"))

	// Verify procs
	assert.Len(t, cfg.Processors, 2)
	assert.Contains(t, cfg.Processors, id.NewID("nop"))
	assert.Contains(t, cfg.Processors, id.NewIDWithName("nop", "myprocessor"))

	// Verify service.
	require.Len(t, cfg.Service.Extensions, 1)
	assert.Contains(t, cfg.Service.Extensions, id.NewID("nop"))
	require.Len(t, cfg.Service.Pipelines, 1)
	assert.Equal(t,
		&config.Pipeline{
			Receivers:  []id.ID{id.NewID("nop")},
			Processors: []id.ID{id.NewID("nop")},
			Exporters:  []id.ID{id.NewID("nop")},
		},
		cfg.Service.Pipelines[id.NewID("traces")],
		"Did not load pipeline config correctly")
}

func TestLoadConfigAndValidate(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	cfgValidate, errValidate := LoadConfigAndValidate(filepath.Join("testdata", "config.yaml"), factories)
	require.NoError(t, errValidate)

	cfg, errLoad := LoadConfig(filepath.Join("testdata", "config.yaml"), factories)
	require.NoError(t, errLoad)

	assert.Equal(t, cfg, cfgValidate)
}
