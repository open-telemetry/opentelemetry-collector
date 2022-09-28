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

package json

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

func TestReadEnumValue(t *testing.T) {
	valueMap := map[string]int32{
		"undefined": 0,
		"foo":       1,
		"bar":       2,
	}
	tests := []struct {
		name    string
		jsonStr string
		want    int32
		wantErr bool
	}{
		{
			name:    "foo string",
			jsonStr: "\"foo\"\n",
			want:    1,
		},
		{
			name:    "foo number",
			jsonStr: "1\n",
			want:    1,
		},
		{
			name:    "unknown number",
			jsonStr: "5\n",
			want:    5,
		},
		{
			name:    "bar string",
			jsonStr: "\"bar\"\n",
			want:    2,
		},
		{
			name:    "bar number",
			jsonStr: "2\n",
			want:    2,
		},
		{
			name:    "unknown string",
			jsonStr: "\"baz\"\n",
			wantErr: true,
		},
		{
			name:    "wrong type",
			jsonStr: "true",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.jsonStr))
			defer jsoniter.ConfigFastest.ReturnIterator(iter)
			val := ReadEnumValue(iter, valueMap)
			if tt.wantErr {
				assert.Error(t, iter.Error)
				return
			}
			assert.NoError(t, iter.Error)
			assert.Equal(t, tt.want, val)
		})
	}
}
