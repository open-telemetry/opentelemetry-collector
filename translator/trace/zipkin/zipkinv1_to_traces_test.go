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
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSingleJSONV1BatchToTraces(t *testing.T) {
	blob, err := ioutil.ReadFile("./testdata/zipkin_v1_single_batch.json")
	require.NoError(t, err, "Failed to load test data")

	got, err := V1JSONBatchToInternalTraces(blob, false)
	require.NoError(t, err, "Failed to translate zipkinv1 to OC proto")

	assert.Equal(t, 5, got.SpanCount())
}

func TestErrorSpanToTraces(t *testing.T) {
	blob, err := ioutil.ReadFile("./testdata/zipkin_v1_error_batch.json")
	require.NoError(t, err, "Failed to load test data")

	td, err := V1JSONBatchToInternalTraces(blob, false)
	assert.Error(t, err, "Should have generated error")
	assert.NotNil(t, td)
}
