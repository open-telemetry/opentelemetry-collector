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

package overwritepropertiesconverter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
)

func TestOverwritePropertiesConverter_Empty(t *testing.T) {
	pmp := New(nil)
	conf := confmap.NewFromStringMap(map[string]interface{}{"foo": "bar"})
	assert.NoError(t, pmp.Convert(context.Background(), conf))
	assert.Equal(t, map[string]interface{}{"foo": "bar"}, conf.ToStringMap())
}

func TestOverwritePropertiesConverter(t *testing.T) {
	props := []string{
		"processors.batch.timeout=2s",
		"processors.batch/foo.timeout=3s",
		"receivers.otlp.protocols.grpc.endpoint=localhost:1818",
		"exporters.kafka.brokers=foo:9200,foo2:9200",
	}

	pmp := New(props)
	conf := confmap.New()
	require.NoError(t, pmp.Convert(context.Background(), conf))
	keys := conf.AllKeys()
	assert.Len(t, keys, 4)
	assert.Equal(t, "2s", conf.Get("processors::batch::timeout"))
	assert.Equal(t, "3s", conf.Get("processors::batch/foo::timeout"))
	assert.Equal(t, "foo:9200,foo2:9200", conf.Get("exporters::kafka::brokers"))
	assert.Equal(t, "localhost:1818", conf.Get("receivers::otlp::protocols::grpc::endpoint"))
}

func TestOverwritePropertiesConverter_InvalidProperty(t *testing.T) {
	pmp := New([]string{"=2s"})
	conf := confmap.New()
	assert.Error(t, pmp.Convert(context.Background(), conf))
}
