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

package spanprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configcheck"
	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
)

func TestFactory_Type(t *testing.T) {
	factory := &Factory{}
	assert.Equal(t, factory.Type(), typeStr)
}

func TestFactory_CreateDefaultConfig(t *testing.T) {
	factory := &Factory{}
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, configcheck.ValidateConfig(cfg))

	// Check the values of the default configuration.
	assert.NotNil(t, cfg)
	assert.Equal(t, typeStr, cfg.Type())
	assert.Equal(t, typeStr, cfg.Name())
}

func TestFactory_CreateTraceProcessor(t *testing.T) {
	factory := &Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)

	// Name.FromAttributes field needs to be set for the configuration to be valid.
	oCfg.Rename.FromAttributes = []string{"test-key"}
	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), oCfg)

	require.Nil(t, err)
	assert.NotNil(t, tp)
}

// TestFactory_CreateTraceProcessor_InvalidConfig ensures the default configuration
// returns an error.
func TestFactory_CreateTraceProcessor_InvalidConfig(t *testing.T) {
	factory := &Factory{}
	cfg := factory.CreateDefaultConfig()

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	require.Nil(t, tp)
	assert.Equal(t, err, errMissingRequiredField)
}

func TestFactory_CreateMetricProcessor(t *testing.T) {
	factory := &Factory{}
	cfg := factory.CreateDefaultConfig()

	mp, err := factory.CreateMetricsProcessor(zap.NewNop(), nil, cfg)
	require.Nil(t, mp)
	assert.Equal(t, err, configerror.ErrDataTypeIsNotSupported)
}
