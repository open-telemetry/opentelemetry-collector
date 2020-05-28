// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resourcedetection

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/api/kv"
)

func TestDecodeLabels(t *testing.T) {
	cases := []struct {
		encoded    string
		wantLabels []kv.KeyValue
		wantFail   bool
	}{
		{
			encoded:    `example.org/test-1="test $ \"" ,  Abc="Def"`,
			wantLabels: []kv.KeyValue{kv.String("example.org/test-1", "test $ \""), kv.String("Abc", "Def")},
		}, {
			encoded:    `single="key"`,
			wantLabels: []kv.KeyValue{kv.String("single", "key")},
		},
		{encoded: `invalid-char-ü="test"`, wantFail: true},
		{encoded: `invalid-char="ü-test"`, wantFail: true},
		{encoded: `missing="trailing-quote`, wantFail: true},
		{encoded: `missing=leading-quote"`, wantFail: true},
		{encoded: `extra="chars", a`, wantFail: true},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			res, err := decodeLabels(c.encoded)

			if c.wantFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, c.wantLabels, res)
		})
	}
}
