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

package spanprocessor

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-service/config"
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
)

func TestLoadConfig(t *testing.T) {
	receivers, processors, exporters, err := config.ExampleComponents()
	assert.NoError(t, err)

	factory := &Factory{}
	processors[typeStr] = factory

	config, err := config.LoadConfigFile(
		t,
		path.Join(".", "testdata", "config.yaml"),
		receivers,
		processors,
		exporters)

	assert.Nil(t, err)
	assert.NotNil(t, config)

	p0 := config.Processors["span/custom"]
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

	p1 := config.Processors["span/no-separator"]
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
}
