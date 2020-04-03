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

package aggregateprocessor

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/config"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
)

func TestLoadingConifg(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.Nil(t, err)

	factory := &Factory{}
	factories.Processors[typeStr] = factory
	config, err := config.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	assert.Nil(t, err)
	require.NotNil(t, config)

	p0 := config.Processors["aggregate"]
	assert.Equal(t, p0, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "aggregate",
			TypeVal: typeStr,
		},
		PeerDiscoveryDNSName: "otel-agent.my-ns.svc.cluster.local",
		PeerPort:             55678,
	})
}
