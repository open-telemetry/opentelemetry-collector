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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPropertiesProvider(t *testing.T) {
	setFlagStr := []string{
		"processors.batch.timeout=2s",
		"processors.batch/foo.timeout=3s",
		"receivers.otlp.protocols.grpc.endpoint=localhost:1818",
		"exporters.kafka.brokers=foo:9200,foo2:9200",
	}

	pmp := NewPropertiesMapProvider(setFlagStr)
	cp, err := pmp.Get(context.Background())
	require.NoError(t, err)
	keys := cp.AllKeys()
	assert.Len(t, keys, 4)
	assert.Equal(t, "2s", cp.Get("processors::batch::timeout"))
	assert.Equal(t, "3s", cp.Get("processors::batch/foo::timeout"))
	assert.Equal(t, "foo:9200,foo2:9200", cp.Get("exporters::kafka::brokers"))
	assert.Equal(t, "localhost:1818", cp.Get("receivers::otlp::protocols::grpc::endpoint"))
	require.NoError(t, pmp.Close(context.Background()))
}

func TestPropertiesProvider_empty(t *testing.T) {
	pmp := NewPropertiesMapProvider(nil)
	cp, err := pmp.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, len(cp.AllKeys()))
	require.NoError(t, pmp.Close(context.Background()))
}
