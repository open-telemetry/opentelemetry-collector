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

package pprofextension

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configcheck"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/testutils"
)

func TestFactory_Type(t *testing.T) {
	factory := Factory{}
	require.Equal(t, typeStr, factory.Type())
}

func TestFactory_CreateDefaultConfig(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, &Config{
		ExtensionSettings: configmodels.ExtensionSettings{
			NameVal: typeStr,
			TypeVal: typeStr,
		},
		Endpoint: "localhost:1777",
	},
		cfg)

	assert.NoError(t, configcheck.ValidateConfig(cfg))
	ext, err := factory.CreateExtension(zap.NewNop(), cfg)
	require.NoError(t, err)
	require.NotNil(t, ext)

	// Restore instance tracking from factory, for other tests.
	atomic.StoreInt32(&instanceState, instanceNotCreated)
}

func TestFactory_CreateExtension(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Endpoint = testutils.GetAvailableLocalAddress(t)

	ext, err := factory.CreateExtension(zap.NewNop(), cfg)
	require.NoError(t, err)
	require.NotNil(t, ext)

	// Restore instance tracking from factory, for other tests.
	atomic.StoreInt32(&instanceState, instanceNotCreated)
}

func TestFactory_CreateExtensionOnlyOnce(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Endpoint = testutils.GetAvailableLocalAddress(t)

	logger := zap.NewNop()
	ext, err := factory.CreateExtension(logger, cfg)
	require.NoError(t, err)
	require.NotNil(t, ext)

	ext1, err := factory.CreateExtension(logger, cfg)
	require.Error(t, err)
	require.Nil(t, ext1)

	// Restore instance tracking from factory, for other tests.
	atomic.StoreInt32(&instanceState, instanceNotCreated)
}
