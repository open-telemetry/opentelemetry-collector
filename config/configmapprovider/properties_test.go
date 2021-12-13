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

package configmapprovider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPropertiesProvider(t *testing.T) {
	t.Parallel()

	setFlagStr := []string{
		"processors.batch.timeout=2s",
		"processors.batch/foo.timeout=3s",
		"receivers.otlp.protocols.grpc.endpoint=localhost:1818",
		"exporters.kafka.brokers=foo:9200,foo2:9200",
	}

	pmp := NewProperties(setFlagStr)
	retr, err := pmp.Retrieve(context.Background(), nil)
	require.NoError(t, err)
	cfgMap, err := retr.Get(context.Background())
	require.NoError(t, err)
	keys := cfgMap.AllKeys()
	assert.Len(t, keys, 4)
	assert.Equal(t, "2s", cfgMap.Get("processors::batch::timeout"))
	assert.Equal(t, "3s", cfgMap.Get("processors::batch/foo::timeout"))
	assert.Equal(t, "foo:9200,foo2:9200", cfgMap.Get("exporters::kafka::brokers"))
	assert.Equal(t, "localhost:1818", cfgMap.Get("receivers::otlp::protocols::grpc::endpoint"))
	require.NoError(t, pmp.Shutdown(context.Background()))
}

func TestPropertiesProvider_empty(t *testing.T) {
	t.Parallel()

	pmp := NewProperties(nil)
	retr, err := pmp.Retrieve(context.Background(), nil)
	require.NoError(t, err)
	cfgMap, err := retr.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, len(cfgMap.AllKeys()))
	require.NoError(t, pmp.Shutdown(context.Background()))
}
