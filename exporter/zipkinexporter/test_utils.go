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

package zipkinexporter

import (
	"encoding/json"
	"testing"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/stretchr/testify/require"
)

func unmarshalZipkinSpanArrayToMap(t *testing.T, jsonStr string) map[zipkinmodel.ID]*zipkinmodel.SpanModel {
	var i interface{}

	err := json.Unmarshal([]byte(jsonStr), &i)
	require.NoError(t, err)

	results := make(map[zipkinmodel.ID]*zipkinmodel.SpanModel)

	switch x := i.(type) {
	case []interface{}:
		for _, j := range x {
			span := jsonToSpan(t, j)
			results[span.ID] = span
		}
	default:
		span := jsonToSpan(t, x)
		results[span.ID] = span
	}
	return results
}

func jsonToSpan(t *testing.T, j interface{}) *zipkinmodel.SpanModel {
	b, err := json.Marshal(j)
	require.NoError(t, err)
	span := &zipkinmodel.SpanModel{}
	err = span.UnmarshalJSON(b)
	require.NoError(t, err)
	return span
}
