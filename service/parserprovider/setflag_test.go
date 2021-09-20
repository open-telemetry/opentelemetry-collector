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
	"context"
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configparser"
)

func TestSetFlags(t *testing.T) {
	flags := new(flag.FlagSet)
	Flags(flags)
	err := flags.Parse([]string{
		"--set=processors.batch.timeout=2s",
		"--set=processors.batch/foo.timeout=3s",
		"--set=receivers.otlp.protocols.grpc.endpoint=localhost:1818",
		"--set=exporters.kafka.brokers=foo:9200,foo2:9200",
	})
	require.NoError(t, err)

	sfl := NewSetFlag(new(testProvider))
	var ret Retrieved
	ret, err = sfl.Retrieve(context.Background())
	require.NoError(t, err)
	var cfgMap *configparser.ConfigMap
	cfgMap, err = ret.Get()
	require.NoError(t, err)

	assert.Len(t, cfgMap.AllKeys(), 4)
	assert.Equal(t, "2s", cfgMap.Get("processors::batch::timeout"))
	assert.Equal(t, "3s", cfgMap.Get("processors::batch/foo::timeout"))
	assert.Equal(t, "foo:9200,foo2:9200", cfgMap.Get("exporters::kafka::brokers"))
	assert.Equal(t, "localhost:1818", cfgMap.Get("receivers::otlp::protocols::grpc::endpoint"))
}

func TestSetFlags_empty(t *testing.T) {
	flags := new(flag.FlagSet)
	Flags(flags)

	sfl := NewSetFlag(new(testProvider))
	ret, err := sfl.Retrieve(context.Background())
	require.NoError(t, err)
	var cfgMap *configparser.ConfigMap
	cfgMap, err = ret.Get()
	require.NoError(t, err)
	assert.Equal(t, generateConfig(), cfgMap)
}

func generateConfig() *configparser.ConfigMap {
	return configparser.NewConfigMapFromStringMap(map[string]interface{}{
		"processors::batch::timeout":                 "1s",
		"processors::batch/foo::timeout":             "2s",
		"receivers::otlp::protocols::grpc::endpoint": "localhost:1717",
		"exporters::kafka::brokers":                  "foo:9200,foo2:8200",
	})
}

type testProvider struct{}

func (el *testProvider) Retrieve(context.Context) (Retrieved, error) {
	return &simpleRetrieved{confMap: generateConfig()}, nil
}

func (el *testProvider) Close(context.Context) error {
	return nil
}
