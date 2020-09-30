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

package spanprocessor

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/internal/processor/filterconfig"
	"go.opentelemetry.io/collector/internal/processor/filterset"
)

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Processors[typeStr] = factory

	cfg, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	p0 := cfg.Processors["span/custom"]
	assert.Equal(t, p0, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: "span/custom",
		},
		Rename: Name{
			FromAttributes: []string{"db.svc", "operation", "id"},
			Separator:      "::",
		},
	})

	p1 := cfg.Processors["span/no-separator"]
	assert.Equal(t, p1, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: "span/no-separator",
		},
		Rename: Name{
			FromAttributes: []string{"db.svc", "operation", "id"},
			Separator:      "",
		},
	})

	p2 := cfg.Processors["span/to_attributes"]
	assert.Equal(t, p2, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: "span/to_attributes",
		},
		Rename: Name{
			ToAttributes: &ToAttributes{
				Rules: []string{`^\/api\/v1\/document\/(?P<documentId>.*)\/update$`},
			},
		},
	})

	p3 := cfg.Processors["span/includeexclude"]
	assert.Equal(t, p3, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: "span/includeexclude",
		},
		MatchConfig: filterconfig.MatchConfig{
			Include: &filterconfig.MatchProperties{
				Config:    *createMatchConfig(filterset.Regexp),
				Services:  []string{`banks`},
				SpanNames: []string{"^(.*?)/(.*?)$"},
			},
			Exclude: &filterconfig.MatchProperties{
				Config:    *createMatchConfig(filterset.Strict),
				SpanNames: []string{`donot/change`},
			},
		},
		Rename: Name{
			ToAttributes: &ToAttributes{
				Rules: []string{`(?P<operation_website>.*?)$`},
			},
		},
	})
}

func createMatchConfig(matchType filterset.MatchType) *filterset.Config {
	return &filterset.Config{
		MatchType: matchType,
	}
}
