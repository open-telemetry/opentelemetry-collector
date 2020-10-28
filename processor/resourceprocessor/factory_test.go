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

package resourceprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, configcheck.ValidateConfig(cfg))
	assert.NotNil(t, cfg)
}

func TestCreateProcessor(t *testing.T) {
	factory := NewFactory()
	cfg := &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: "resource",
			NameVal: "resource",
		},
		AttributesActions: []processorhelper.ActionKeyValue{
			{Key: "cloud.zone", Value: "zone-1", Action: processorhelper.UPSERT},
		},
	}

	tp, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewTracesNop())
	assert.NoError(t, err)
	assert.NotNil(t, tp)

	mp, err := factory.CreateMetricsProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewMetricsNop())
	assert.NoError(t, err)
	assert.NotNil(t, mp)
}

func TestInvalidEmptyActions(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	_, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewTracesNop())
	assert.Error(t, err)

	_, err = factory.CreateMetricsProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewMetricsNop())
	assert.Error(t, err)
}

func TestInvalidAttributeActions(t *testing.T) {
	factory := NewFactory()
	cfg := &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: "resource",
			NameVal: "resource",
		},
		AttributesActions: []processorhelper.ActionKeyValue{
			{Key: "k", Value: "v", Action: "invalid-action"},
		},
	}

	_, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, nil)
	assert.Error(t, err)

	_, err = factory.CreateMetricsProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, nil)
	assert.Error(t, err)
}

func TestDeprecatedConfig(t *testing.T) {
	cfg := &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: "resource",
			NameVal: "resource",
		},
		ResourceType: "host",
		Labels: map[string]string{
			"cloud.zone": "zone-1",
		},
	}

	handleDeprecatedFields(cfg, zap.NewNop())

	assert.EqualValues(t, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: "resource",
			NameVal: "resource",
		},
		ResourceType: "host",
		Labels: map[string]string{
			"cloud.zone": "zone-1",
		},
		AttributesActions: []processorhelper.ActionKeyValue{
			{Key: "opencensus.resourcetype", Value: "host", Action: processorhelper.UPSERT},
			{Key: "cloud.zone", Value: "zone-1", Action: processorhelper.UPSERT},
		},
	}, cfg)
}
