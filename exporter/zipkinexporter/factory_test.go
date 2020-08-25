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

package zipkinexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcheck"
)

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateInstanceViaFactory(t *testing.T) {
	cfg := createDefaultConfig()

	// Default config doesn't have default endpoint so creating from it should
	// fail.
	ze, err := createTraceExporter(context.Background(), component.ExporterCreateParams{Logger: zap.NewNop()}, cfg)
	assert.Error(t, err)
	assert.Nil(t, ze)

	// URL doesn't have a default value so set it directly.
	zeCfg := cfg.(*Config)
	zeCfg.Endpoint = "http://some.location.org:9411/api/v2/spans"
	ze, err = createTraceExporter(context.Background(), component.ExporterCreateParams{Logger: zap.NewNop()}, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, ze)
}
