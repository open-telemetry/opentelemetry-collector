// Copyright The OpenTelemetry Authors
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

package parserprovider

import (
	"flag"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configloader"
	"go.opentelemetry.io/collector/config/configparser"
	"go.opentelemetry.io/collector/processor/attributesprocessor"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/receiver/jaegerreceiver"
	"go.opentelemetry.io/collector/service/defaultcomponents"
)

func TestDefault(t *testing.T) {
	factories, err := defaultcomponents.Components()
	require.NoError(t, err)
	t.Run("unknown_component", func(t *testing.T) {
		flags := new(flag.FlagSet)
		Flags(flags)
		err = flags.Parse([]string{
			"--config=testdata/otelcol-config.yaml",
			"--set=processors.doesnotexist.timeout=2s",
		})
		require.NoError(t, err)
		pl := Default()
		require.NotNil(t, pl)
		var cp *configparser.Parser
		cp, err = pl.Get()
		require.NoError(t, err)
		require.NotNil(t, cp)
		var cfg *config.Config
		cfg, err = configloader.Load(cp, factories)
		require.Error(t, err)
		require.Nil(t, cfg)

	})
	t.Run("component_not_added_to_pipeline", func(t *testing.T) {
		flags := new(flag.FlagSet)
		Flags(flags)
		err = flags.Parse([]string{
			"--config=testdata/otelcol-config.yaml",
			"--set=processors.batch/foo.timeout=2s",
		})
		require.NoError(t, err)
		pl := Default()
		require.NotNil(t, pl)
		var cp *configparser.Parser
		cp, err = pl.Get()
		require.NoError(t, err)
		require.NotNil(t, cp)
		var cfg *config.Config
		cfg, err = configloader.Load(cp, factories)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		err = cfg.Validate()
		require.NoError(t, err)

		var processors []string
		for k := range cfg.Processors {
			processors = append(processors, k.String())
		}
		sort.Strings(processors)
		// batch/foo is not added to the pipeline
		assert.Equal(t, []string{"attributes", "batch", "batch/foo"}, processors)
		assert.Equal(t, []config.ComponentID{config.NewID("attributes"), config.NewID("batch")}, cfg.Service.Pipelines["traces"].Processors)
	})
	t.Run("ok", func(t *testing.T) {
		flags := new(flag.FlagSet)
		Flags(flags)
		err = flags.Parse([]string{
			"--config=testdata/otelcol-config.yaml",
			"--set=processors.batch.timeout=2s",
			// Arrays are overridden and object arrays cannot be indexed
			// this creates actions array of size 1
			"--set=processors.attributes.actions.key=foo",
			"--set=processors.attributes.actions.value=bar",
			"--set=receivers.jaeger.protocols.grpc.endpoint=localhost:12345",
			"--set=extensions.health_check.endpoint=localhost:8080",
		})
		require.NoError(t, err)
		pl := Default()
		require.NotNil(t, pl)
		var cp *configparser.Parser
		cp, err = pl.Get()
		require.NoError(t, err)
		require.NotNil(t, cp)
		var cfg *config.Config
		cfg, err = configloader.Load(cp, factories)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		err = cfg.Validate()
		require.NoError(t, err)

		assert.Equal(t, 2, len(cfg.Processors))
		batch := cfg.Processors[config.NewID("batch")].(*batchprocessor.Config)
		assert.Equal(t, time.Second*2, batch.Timeout)
		jaeger := cfg.Receivers[config.NewID("jaeger")].(*jaegerreceiver.Config)
		assert.Equal(t, "localhost:12345", jaeger.GRPC.NetAddr.Endpoint)
		attributes := cfg.Processors[config.NewID("attributes")].(*attributesprocessor.Config)
		require.Equal(t, 1, len(attributes.Actions))
		assert.Equal(t, "foo", attributes.Actions[0].Key)
		assert.Equal(t, "bar", attributes.Actions[0].Value)
	})
}

func TestDefault_ComponentDoesNotExist(t *testing.T) {
	factories, err := defaultcomponents.Components()
	require.NoError(t, err)

	flags := new(flag.FlagSet)
	Flags(flags)
	err = flags.Parse([]string{
		"--config=testdata/otelcol-config.yaml",
		"--set=processors.batch.timeout=2s",
		// Arrays are overridden and object arrays cannot be indexed
		// this creates actions array of size 1
		"--set=processors.attributes.actions.key=foo",
		"--set=processors.attributes.actions.value=bar",
		"--set=receivers.jaeger.protocols.grpc.endpoint=localhost:12345",
	})
	require.NoError(t, err)

	pl := Default()
	require.NotNil(t, pl)
	cp, err := pl.Get()
	require.NoError(t, err)
	require.NotNil(t, cp)
	cfg, err := configloader.Load(cp, factories)
	require.NoError(t, err)
	require.NotNil(t, cfg)
}
