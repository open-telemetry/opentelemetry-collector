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

package batchprocessor

import (
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtest"
)

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Processors[typeStr] = factory
	cfg, err := configtest.LoadConfigAndValidate(path.Join(".", "testdata", "config.yaml"), factories)

	require.Nil(t, err)
	require.NotNil(t, cfg)

	p0 := cfg.Processors[config.NewID(typeStr)]
	assert.Equal(t, p0, factory.CreateDefaultConfig())

	p1 := cfg.Processors[config.NewIDWithName(typeStr, "2")]

	timeout := time.Second * 10
	sendBatchSize := uint32(10000)
	sendBatchMaxSize := uint32(11000)

	assert.Equal(t, p1,
		&Config{
			ProcessorSettings: config.NewProcessorSettings(config.NewIDWithName(typeStr, "2")),
			SendBatchSize:     sendBatchSize,
			SendBatchMaxSize:  sendBatchMaxSize,
			Timeout:           timeout,
		})
}

func TestValidateConfig_DefaultBatchMaxSize(t *testing.T) {
	cfg := &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewIDWithName(typeStr, "2")),
		SendBatchSize:     100,
		SendBatchMaxSize:  0,
	}
	assert.NoError(t, cfg.Validate())
}

func TestValidateConfig_ValidBatchSizes(t *testing.T) {
	cfg := &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewIDWithName(typeStr, "2")),
		SendBatchSize:     100,
		SendBatchMaxSize:  1000,
	}
	assert.NoError(t, cfg.Validate())

}

func TestValidateConfig_InvalidBatchSize(t *testing.T) {
	cfg := &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewIDWithName(typeStr, "2")),
		SendBatchSize:     1000,
		SendBatchMaxSize:  100,
	}
	assert.Error(t, cfg.Validate())
}
