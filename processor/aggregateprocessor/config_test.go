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

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configtest"
)

func TestLoadingConifg(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.Nil(t, err)

    factory := NewFactory()
	factories.Processors[factory.Type()] = factory

	cfg, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)
	require.Nil(t, err)
	require.NotNil(t, cfg)

	assert.Nil(t, err)
	require.NotNil(t, cfg)

	p0 := cfg.Processors["aggregate"]
	assert.Equal(t, p0, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: "aggregate",
			TypeVal: "aggregate",
		},
		PeerDiscoveryDNSName: "otel-agent.my-ns.svc.cluster.local",
		PeerPort:             55678,
	})
}
