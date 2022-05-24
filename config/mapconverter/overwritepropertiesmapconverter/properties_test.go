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

package overwritepropertiesmapconverter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config"
)

func TestOverwritePropertiesConverter_Empty(t *testing.T) {
	pmp := New(nil)
	cfgMap := config.NewMapFromStringMap(map[string]interface{}{"foo": "bar"})
	assert.NoError(t, pmp.Convert(context.Background(), cfgMap))
	assert.Equal(t, map[string]interface{}{"foo": "bar"}, cfgMap.ToStringMap())
}

func TestOverwritePropertiesConverter(t *testing.T) {
	props := []string{
		"processors.batch.timeout=2s",
		"processors.batch/foo.timeout=3s",
		"receivers.otlp.protocols.grpc.endpoint=localhost:1818",
		"exporters.kafka.brokers=foo:9200,foo2:9200",
	}

	pmp := New(props)
	cfgMap := config.NewMap()
	require.NoError(t, pmp.Convert(context.Background(), cfgMap))
	keys := cfgMap.AllKeys()
	assert.Len(t, keys, 4)
	assert.Equal(t, "2s", cfgMap.Get("processors::batch::timeout"))
	assert.Equal(t, "3s", cfgMap.Get("processors::batch/foo::timeout"))
	assert.Equal(t, "foo:9200,foo2:9200", cfgMap.Get("exporters::kafka::brokers"))
	assert.Equal(t, "localhost:1818", cfgMap.Get("receivers::otlp::protocols::grpc::endpoint"))
}

func TestOverwritePropertiesConverter_InvalidProperty(t *testing.T) {
	pmp := New([]string{"=2s"})
	cfgMap := config.NewMap()
	assert.Error(t, pmp.Convert(context.Background(), cfgMap))
}
