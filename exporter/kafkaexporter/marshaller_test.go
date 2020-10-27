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

package kafkaexporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultTracesMarshallers(t *testing.T) {
	expectedEncodings := []string{
		"otlp_proto",
		"jaeger_proto",
		"jaeger_json",
	}
	marshallers := tracesMarshallers()
	assert.Equal(t, len(expectedEncodings), len(marshallers))
	for _, e := range expectedEncodings {
		t.Run(e, func(t *testing.T) {
			m, ok := marshallers[e]
			require.True(t, ok)
			assert.NotNil(t, m)
		})
	}
}

func TestDefaultMetricsMarshallers(t *testing.T) {
	expectedEncodings := []string{
		"otlp_proto",
	}
	marshallers := metricsMarshallers()
	assert.Equal(t, len(expectedEncodings), len(marshallers))
	for _, e := range expectedEncodings {
		t.Run(e, func(t *testing.T) {
			m, ok := marshallers[e]
			require.True(t, ok)
			assert.NotNil(t, m)
		})
	}
}
