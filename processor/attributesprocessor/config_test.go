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

package attributesprocessor

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/internal/processor/filterconfig"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

func TestLoadingConfig(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Processors[typeStr] = factory
	cfg, err := configtest.LoadConfigAndValidate(path.Join(".", "testdata", "config.yaml"), factories)
	assert.NoError(t, err)
	require.NotNil(t, cfg)

	p0 := cfg.Processors[config.NewIDWithName(typeStr, "insert")]
	assert.Equal(t, p0, &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewIDWithName(typeStr, "insert")),
		Settings: processorhelper.Settings{
			Actions: []processorhelper.ActionKeyValue{
				{Key: "attribute1", Value: 123, Action: processorhelper.INSERT},
				{Key: "string key", FromAttribute: "anotherkey", Action: processorhelper.INSERT},
			},
		},
	})

	p1 := cfg.Processors[config.NewIDWithName(typeStr, "update")]
	assert.Equal(t, p1, &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewIDWithName(typeStr, "update")),
		Settings: processorhelper.Settings{
			Actions: []processorhelper.ActionKeyValue{
				{Key: "boo", FromAttribute: "foo", Action: processorhelper.UPDATE},
				{Key: "db.secret", Value: "redacted", Action: processorhelper.UPDATE},
			},
		},
	})

	p2 := cfg.Processors[config.NewIDWithName(typeStr, "upsert")]
	assert.Equal(t, p2, &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewIDWithName(typeStr, "upsert")),
		Settings: processorhelper.Settings{
			Actions: []processorhelper.ActionKeyValue{
				{Key: "region", Value: "planet-earth", Action: processorhelper.UPSERT},
				{Key: "new_user_key", FromAttribute: "user_key", Action: processorhelper.UPSERT},
			},
		},
	})

	p3 := cfg.Processors[config.NewIDWithName(typeStr, "delete")]
	assert.Equal(t, p3, &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewIDWithName(typeStr, "delete")),
		Settings: processorhelper.Settings{
			Actions: []processorhelper.ActionKeyValue{
				{Key: "credit_card", Action: processorhelper.DELETE},
				{Key: "duplicate_key", Action: processorhelper.DELETE},
			},
		},
	})

	p4 := cfg.Processors[config.NewIDWithName(typeStr, "hash")]
	assert.Equal(t, p4, &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewIDWithName(typeStr, "hash")),
		Settings: processorhelper.Settings{
			Actions: []processorhelper.ActionKeyValue{
				{Key: "user.email", Action: processorhelper.HASH},
			},
		},
	})

	p5 := cfg.Processors[config.NewIDWithName(typeStr, "excludemulti")]
	assert.Equal(t, p5, &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewIDWithName(typeStr, "excludemulti")),
		MatchConfig: filterconfig.MatchConfig{
			Exclude: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{"svcA", "svcB"},
				Attributes: []filterconfig.Attribute{
					{Key: "env", Value: "dev"},
					{Key: "test_request"},
				},
			},
		},
		Settings: processorhelper.Settings{
			Actions: []processorhelper.ActionKeyValue{
				{Key: "credit_card", Action: processorhelper.DELETE},
				{Key: "duplicate_key", Action: processorhelper.DELETE},
			},
		},
	})

	p6 := cfg.Processors[config.NewIDWithName(typeStr, "includeservices")]
	assert.Equal(t, p6, &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewIDWithName(typeStr, "includeservices")),
		MatchConfig: filterconfig.MatchConfig{
			Include: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Regexp),
				Services: []string{"auth.*", "login.*"},
			},
		},
		Settings: processorhelper.Settings{
			Actions: []processorhelper.ActionKeyValue{
				{Key: "credit_card", Action: processorhelper.DELETE},
				{Key: "duplicate_key", Action: processorhelper.DELETE},
			},
		},
	})

	p7 := cfg.Processors[config.NewIDWithName(typeStr, "selectiveprocessing")]
	assert.Equal(t, p7, &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewIDWithName(typeStr, "selectiveprocessing")),
		MatchConfig: filterconfig.MatchConfig{
			Include: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{"svcA", "svcB"},
			},
			Exclude: &filterconfig.MatchProperties{
				Config: *createConfig(filterset.Strict),
				Attributes: []filterconfig.Attribute{
					{Key: "redact_trace", Value: false},
				},
			},
		},
		Settings: processorhelper.Settings{
			Actions: []processorhelper.ActionKeyValue{
				{Key: "credit_card", Action: processorhelper.DELETE},
				{Key: "duplicate_key", Action: processorhelper.DELETE},
			},
		},
	})

	p8 := cfg.Processors[config.NewIDWithName(typeStr, "complex")]
	assert.Equal(t, p8, &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewIDWithName(typeStr, "complex")),
		Settings: processorhelper.Settings{
			Actions: []processorhelper.ActionKeyValue{
				{Key: "operation", Value: "default", Action: processorhelper.INSERT},
				{Key: "svc.operation", FromAttribute: "operation", Action: processorhelper.UPSERT},
				{Key: "operation", Action: processorhelper.DELETE},
			},
		},
	})

	p9 := cfg.Processors[config.NewIDWithName(typeStr, "example")]
	assert.Equal(t, p9, &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewIDWithName(typeStr, "example")),
		Settings: processorhelper.Settings{
			Actions: []processorhelper.ActionKeyValue{
				{Key: "db.table", Action: processorhelper.DELETE},
				{Key: "redacted_span", Value: true, Action: processorhelper.UPSERT},
				{Key: "copy_key", FromAttribute: "key_original", Action: processorhelper.UPDATE},
				{Key: "account_id", Value: 2245, Action: processorhelper.INSERT},
				{Key: "account_password", Action: processorhelper.DELETE},
			},
		},
	})

	p10 := cfg.Processors[config.NewIDWithName(typeStr, "regexp")]
	assert.Equal(t, p10, &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewIDWithName(typeStr, "regexp")),
		MatchConfig: filterconfig.MatchConfig{
			Include: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Regexp),
				Services: []string{"auth.*"},
			},
			Exclude: &filterconfig.MatchProperties{
				Config:    *createConfig(filterset.Regexp),
				SpanNames: []string{"login.*"},
			},
		},
		Settings: processorhelper.Settings{
			Actions: []processorhelper.ActionKeyValue{
				{Key: "password", Action: processorhelper.UPDATE, Value: "obfuscated"},
				{Key: "token", Action: processorhelper.DELETE},
			},
		},
	})

}
