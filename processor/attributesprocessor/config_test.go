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

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/internal/processor/filterspan"
)

func TestLoadingConifg(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.NoError(t, err)

	factory := &Factory{}
	factories.Processors[typeStr] = factory
	config, err := config.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	assert.NoError(t, err)
	require.NotNil(t, config)

	p0 := config.Processors[configmodels.MustParseEntityName("attributes/insert")]
	assert.Equal(t, p0, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/insert",
			TypeVal: typeStr,
		},
		Actions: []ActionKeyValue{
			{Key: "attribute1", Value: 123, Action: INSERT},
			{Key: "string key", FromAttribute: "anotherkey", Action: INSERT},
		},
	})

	p1 := config.Processors[configmodels.MustParseEntityName("attributes/update")]
	assert.Equal(t, p1, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/update",
			TypeVal: typeStr,
		},
		Actions: []ActionKeyValue{
			{Key: "boo", FromAttribute: "foo", Action: UPDATE},
			{Key: "db.secret", Value: "redacted", Action: UPDATE},
		},
	})

	p2 := config.Processors[configmodels.MustParseEntityName("attributes/upsert")]
	assert.Equal(t, p2, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/upsert",
			TypeVal: typeStr,
		},
		Actions: []ActionKeyValue{
			{Key: "region", Value: "planet-earth", Action: UPSERT},
			{Key: "new_user_key", FromAttribute: "user_key", Action: UPSERT},
		},
	})

	p3 := config.Processors[configmodels.MustParseEntityName("attributes/delete")]
	assert.Equal(t, p3, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/delete",
			TypeVal: typeStr,
		},
		Actions: []ActionKeyValue{
			{Key: "credit_card", Action: DELETE},
			{Key: "duplicate_key", Action: DELETE},
		},
	})

	p4 := config.Processors[configmodels.MustParseEntityName("attributes/hash")]
	assert.Equal(t, p4, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/hash",
			TypeVal: typeStr,
		},
		Actions: []ActionKeyValue{
			{Key: "user.email", Action: HASH},
		},
	})

	p5 := config.Processors[configmodels.MustParseEntityName("attributes/excludemulti")]
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
		Actions: []ActionKeyValue{
			{Key: "credit_card", Action: DELETE},
			{Key: "duplicate_key", Action: DELETE},
		},
	})

	p6 := config.Processors[configmodels.MustParseEntityName("attributes/includeservices")]
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
		Actions: []ActionKeyValue{
			{Key: "credit_card", Action: DELETE},
			{Key: "duplicate_key", Action: DELETE},
		},
	})

	p7 := config.Processors[configmodels.MustParseEntityName("attributes/selectiveprocessing")]
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
		Actions: []ActionKeyValue{
			{Key: "credit_card", Action: DELETE},
			{Key: "duplicate_key", Action: DELETE},
		},
	})

	p8 := config.Processors[configmodels.MustParseEntityName("attributes/complex")]
	assert.Equal(t, p8, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/complex",
			TypeVal: typeStr,
		},
		Actions: []ActionKeyValue{
			{Key: "operation", Value: "default", Action: INSERT},
			{Key: "svc.operation", FromAttribute: "operation", Action: UPSERT},
			{Key: "operation", Action: DELETE},
		},
	})

	p9 := config.Processors[configmodels.MustParseEntityName("attributes/example")]
	assert.Equal(t, p9, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "attributes/example",
			TypeVal: typeStr,
		},
		Actions: []ActionKeyValue{
			{Key: "db.table", Action: DELETE},
			{Key: "redacted_span", Value: true, Action: UPSERT},
			{Key: "copy_key", FromAttribute: "key_original", Action: UPDATE},
			{Key: "account_id", Value: 2245, Action: INSERT},
			{Key: "account_password", Action: DELETE},
		},
	})

	p10 := config.Processors[configmodels.MustParseEntityName("attributes/regexp")]
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
		Actions: []ActionKeyValue{
			{Key: "password", Action: UPDATE, Value: "obfuscated"},
			{Key: "token", Action: DELETE},
		},
	})

}
