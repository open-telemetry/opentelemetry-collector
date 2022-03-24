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

package otlpgrpc

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var _ json.Unmarshaler = MetricsResponse{}
var _ json.Marshaler = MetricsResponse{}

var _ json.Unmarshaler = MetricsRequest{}
var _ json.Marshaler = MetricsRequest{}

var metricsRequestJSON = []byte(`
	{
	  "resourceMetrics": [
		{
          "resource": {},
		  "instrumentationLibraryMetrics": [
			{
              "instrumentationLibrary": {},
			  "metrics": [
				{
				  "name": "test_metric"
				}
			  ]
			}
		  ]
		}
	  ]
	}`)

func TestMetricsRequestJSON(t *testing.T) {
	mr := NewMetricsRequest()
	assert.NoError(t, mr.UnmarshalJSON(metricsRequestJSON))
	assert.Equal(t, "test_metric", mr.Metrics().ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Name())

	got, err := mr.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, strings.Join(strings.Fields(string(metricsRequestJSON)), ""), string(got))
}
