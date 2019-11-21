// Copyright 2019 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resourceprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configcheck"
)

func TestCreateDefaultConfig(t *testing.T) {
	var factory Factory
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, configcheck.ValidateConfig(cfg))
	assert.NotNil(t, cfg)
}

func TestCreateProcessor(t *testing.T) {
	var factory Factory
	cfg := factory.CreateDefaultConfig()

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), nil, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, tp)

	mp, err := factory.CreateMetricsProcessor(zap.NewNop(), nil, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, mp)
}
