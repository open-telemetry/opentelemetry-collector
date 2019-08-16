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

package span

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/config/configerror"
	"github.com/open-telemetry/opentelemetry-service/exporter/exportertest"
)

func TestFactory_Type(t *testing.T) {
	factory := &Factory{}
	assert.NotNil(t, factory)
	assert.Equal(t, factory.Type(), typeStr)
}

func TestFactory_CreateDefaultConfig(t *testing.T) {
	factory := &Factory{}
	assert.NotNil(t, factory)

	cfg := factory.CreateDefaultConfig()
	// Check the values of the default configuration.
	assert.Equal(t, typeStr, cfg.Type())
	assert.Equal(t, typeStr, cfg.Name())
	assert.NotNil(t, cfg, "failed to create default config")
}

func TestFactory_CreateTraceProcessor(t *testing.T) {
	factory := &Factory{}
	assert.NotNil(t, factory)

	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)

	// Rename.Keys field needs to be set for the configuration to be valid.
	oCfg.Rename.Keys = []string{"test-key"}
	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), oCfg)
	// TODO(ccaraman): Fix this when the processor logic is added and the trace processor
	//  no longer returns an error.
	assert.Nil(t, tp, "should not be able to create trace processor")
	assert.Equal(t, err, configerror.ErrDataTypeIsNotSupported)
}

// TestFactory_CreateTraceProcessor_InvalidConfig ensures the default configuration
// returns an error.
func TestFactory_CreateTraceProcessor_InvalidConfig(t *testing.T) {
	factory := &Factory{}
	assert.NotNil(t, factory)

	cfg := factory.CreateDefaultConfig()
	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	assert.Nil(t, tp, "should not be able to create trace processor")
	assert.Equal(t, err, errMissingRequiredField, "should not be able to create trace processor")
}

func TestFactory_CreateMetricProcessor(t *testing.T) {
	factory := &Factory{}
	assert.NotNil(t, factory)

	cfg := factory.CreateDefaultConfig()
	mp, err := factory.CreateMetricsProcessor(zap.NewNop(), nil, cfg)
	assert.Nil(t, mp)
	assert.Equal(t, err, configerror.ErrDataTypeIsNotSupported, "should not be able to create metrics processor")
}
