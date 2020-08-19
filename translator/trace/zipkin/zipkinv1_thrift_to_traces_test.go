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

package zipkin

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestV1ThriftToTraces(t *testing.T) {
	blob, err := ioutil.ReadFile("./testdata/zipkin_v1_thrift_single_batch.json")
	require.NoError(t, err, "Failed to load test data")

	var ztSpans []*zipkincore.Span
	err = json.Unmarshal(blob, &ztSpans)
	require.NoError(t, err, "Failed to unmarshal json into zipkin v1 thrift")

	got, err := V1ThriftBatchToInternalTraces(ztSpans)
	require.NoError(t, err, "Failed to translate zipkinv1 thrift to OC proto")

	assert.Equal(t, 5, got.SpanCount())
}
