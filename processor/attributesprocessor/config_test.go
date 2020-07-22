// Copyright The OpenTelemetry Authors
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

package attributesprocessor

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/internal/processor/attraction"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/internal/processor/filterspan"
)

func TestLoadingConifg(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Processors[typeStr] = factory
	cfg, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)
	assert.NoError(t, err)
	require.NotNil(t, cfg)

	p0 := cfg.Processors["attributes/insert"]
	assert.Equal(t, p0, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/insert",
			TypeVal: typeStr,
		},
		Settings: attraction.Settings{
			Actions: []attraction.ActionKeyValue{
				{Key: "attribute1", Value: 123, Action: attraction.INSERT},
				{Key: "string key", FromAttribute: "anotherkey", Action: attraction.INSERT},
			},
		},
	})

	p1 := cfg.Processors["attributes/update"]
	assert.Equal(t, p1, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/update",
			TypeVal: typeStr,
		},
		Settings: attraction.Settings{
			Actions: []attraction.ActionKeyValue{
				{Key: "boo", FromAttribute: "foo", Action: attraction.UPDATE},
				{Key: "db.secret", Value: "redacted", Action: attraction.UPDATE},
			},
		},
	})

	p2 := cfg.Processors["attributes/upsert"]
	assert.Equal(t, p2, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/upsert",
			TypeVal: typeStr,
		},
		Settings: attraction.Settings{
			Actions: []attraction.ActionKeyValue{
				{Key: "region", Value: "planet-earth", Action: attraction.UPSERT},
				{Key: "new_user_key", FromAttribute: "user_key", Action: attraction.UPSERT},
			},
		},
	})

	p3 := cfg.Processors["attributes/delete"]
	assert.Equal(t, p3, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/delete",
			TypeVal: typeStr,
		},
		Settings: attraction.Settings{
			Actions: []attraction.ActionKeyValue{
				{Key: "credit_card", Action: attraction.DELETE},
				{Key: "duplicate_key", Action: attraction.DELETE},
			},
		},
	})

	p4 := cfg.Processors["attributes/hash"]
	assert.Equal(t, p4, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/hash",
			TypeVal: typeStr,
		},
		Settings: attraction.Settings{
			Actions: []attraction.ActionKeyValue{
				{Key: "user.email", Action: attraction.HASH},
			},
		},
	})

	p5 := cfg.Processors["attributes/excludemulti"]
	assert.Equal(t, p5, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/excludemulti",
			TypeVal: typeStr,
		},
		MatchConfig: filterspan.MatchConfig{
			Exclude: &filterspan.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{"svcA", "svcB"},
				Attributes: []filterspan.Attribute{
					{Key: "env", Value: "dev"},
					{Key: "test_request"},
				},
			},
		},
		Settings: attraction.Settings{
			Actions: []attraction.ActionKeyValue{
				{Key: "credit_card", Action: attraction.DELETE},
				{Key: "duplicate_key", Action: attraction.DELETE},
			},
		},
	})

	p6 := cfg.Processors["attributes/includeservices"]
	assert.Equal(t, p6, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/includeservices",
			TypeVal: typeStr,
		},
		MatchConfig: filterspan.MatchConfig{
			Include: &filterspan.MatchProperties{
				Config:   *createConfig(filterset.Regexp),
				Services: []string{"auth.*", "login.*"},
			},
		},
		Settings: attraction.Settings{
			Actions: []attraction.ActionKeyValue{
				{Key: "credit_card", Action: attraction.DELETE},
				{Key: "duplicate_key", Action: attraction.DELETE},
			},
		},
	})

	p7 := cfg.Processors["attributes/selectiveprocessing"]
	assert.Equal(t, p7, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/selectiveprocessing",
			TypeVal: typeStr,
		},
		MatchConfig: filterspan.MatchConfig{
			Include: &filterspan.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{"svcA", "svcB"},
			},
			Exclude: &filterspan.MatchProperties{
				Config: *createConfig(filterset.Strict),
				Attributes: []filterspan.Attribute{
					{Key: "redact_trace", Value: false},
				},
			},
		},
		Settings: attraction.Settings{
			Actions: []attraction.ActionKeyValue{
				{Key: "credit_card", Action: attraction.DELETE},
				{Key: "duplicate_key", Action: attraction.DELETE},
			},
		},
	})

	p8 := cfg.Processors["attributes/complex"]
	assert.Equal(t, p8, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/complex",
			TypeVal: typeStr,
		},
		Settings: attraction.Settings{
			Actions: []attraction.ActionKeyValue{
				{Key: "operation", Value: "default", Action: attraction.INSERT},
				{Key: "svc.operation", FromAttribute: "operation", Action: attraction.UPSERT},
				{Key: "operation", Action: attraction.DELETE},
			},
		},
	})

	p9 := cfg.Processors["attributes/example"]
	assert.Equal(t, p9, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/example",
			TypeVal: typeStr,
		},
		Settings: attraction.Settings{
			Actions: []attraction.ActionKeyValue{
				{Key: "db.table", Action: attraction.DELETE},
				{Key: "redacted_span", Value: true, Action: attraction.UPSERT},
				{Key: "copy_key", FromAttribute: "key_original", Action: attraction.UPDATE},
				{Key: "account_id", Value: 2245, Action: attraction.INSERT},
				{Key: "account_password", Action: attraction.DELETE},
			},
		},
	})

	p10 := cfg.Processors["attributes/regexp"]
	assert.Equal(t, p10, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/regexp",
			TypeVal: typeStr,
		},
		MatchConfig: filterspan.MatchConfig{
			Include: &filterspan.MatchProperties{
				Config:   *createConfig(filterset.Regexp),
				Services: []string{"auth.*"},
			},
			Exclude: &filterspan.MatchProperties{
				Config:    *createConfig(filterset.Regexp),
				SpanNames: []string{"login.*"},
			},
		},
		Settings: attraction.Settings{
			Actions: []attraction.ActionKeyValue{
				{Key: "password", Action: attraction.UPDATE, Value: "obfuscated"},
				{Key: "token", Action: attraction.DELETE},
			},
		},
	})

}
