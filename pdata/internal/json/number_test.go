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

func TestReadInt32(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    int32
		wantErr bool
	}{
		{
			name:    "number",
			jsonStr: `1 `,
			want:    1,
		},
		{
			name:    "string",
			jsonStr: `"1"`,
			want:    1,
		},
		{
			name:    "negative number",
			jsonStr: `-1 `,
			want:    -1,
		},
		{
			name:    "negative string",
			jsonStr: `"-1"`,
			want:    -1,
		},
		{
			name:    "wrong string",
			jsonStr: `"3.f14"`,
			wantErr: true,
		},
		{
			name:    "wrong type",
			jsonStr: `true`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.jsonStr))
			defer jsoniter.ConfigFastest.ReturnIterator(iter)
			val := ReadInt32(iter)
			if tt.wantErr {
				assert.Error(t, iter.Error)
				return
			}
			assert.NoError(t, iter.Error)
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestReadUint32(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    uint32
		wantErr bool
	}{
		{
			name:    "number",
			jsonStr: `1 `,
			want:    1,
		},
		{
			name:    "string",
			jsonStr: `"1"`,
			want:    1,
		},
		{
			name:    "negative number",
			jsonStr: `-1 `,
			wantErr: true,
		},
		{
			name:    "negative string",
			jsonStr: `"-1"`,
			wantErr: true,
		},
		{
			name:    "wrong string",
			jsonStr: `"3.f14"`,
			wantErr: true,
		},
		{
			name:    "wrong type",
			jsonStr: `true`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.jsonStr))
			defer jsoniter.ConfigFastest.ReturnIterator(iter)
			val := ReadUint32(iter)
			if tt.wantErr {
				assert.Error(t, iter.Error)
				return
			}
			assert.NoError(t, iter.Error)
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestReadInt64(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    int64
		wantErr bool
	}{
		{
			name:    "number",
			jsonStr: `1 `,
			want:    1,
		},
		{
			name:    "string",
			jsonStr: `"1"`,
			want:    1,
		},
		{
			name:    "negative number",
			jsonStr: `-1 `,
			want:    -1,
		},
		{
			name:    "negative string",
			jsonStr: `"-1"`,
			want:    -1,
		},
		{
			name:    "wrong string",
			jsonStr: `"3.f14"`,
			wantErr: true,
		},
		{
			name:    "wrong type",
			jsonStr: `true`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.jsonStr))
			defer jsoniter.ConfigFastest.ReturnIterator(iter)
			val := ReadInt64(iter)
			if tt.wantErr {
				assert.Error(t, iter.Error)
				return
			}
			assert.NoError(t, iter.Error)
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestReadUint64(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    uint64
		wantErr bool
	}{
		{
			name:    "number",
			jsonStr: `1 `,
			want:    1,
		},
		{
			name:    "string",
			jsonStr: `"1"`,
			want:    1,
		},
		{
			name:    "negative number",
			jsonStr: `-1 `,
			wantErr: true,
		},
		{
			name:    "negative string",
			jsonStr: `"-1"`,
			wantErr: true,
		},
		{
			name:    "wrong string",
			jsonStr: `"3.f14"`,
			wantErr: true,
		},
		{
			name:    "wrong type",
			jsonStr: `true`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.jsonStr))
			defer jsoniter.ConfigFastest.ReturnIterator(iter)
			val := ReadUint64(iter)
			if tt.wantErr {
				assert.Error(t, iter.Error)
				return
			}
			assert.NoError(t, iter.Error)
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestReadFloat64(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    float64
		wantErr bool
	}{
		{
			name:    "number",
			jsonStr: `3.14 `,
			want:    3.14,
		},
		{
			name:    "string",
			jsonStr: `"3.14"`,
			want:    3.14,
		},
		{
			name:    "negative number",
			jsonStr: `-3.14 `,
			want:    -3.14,
		},
		{
			name:    "negative string",
			jsonStr: `"-3.14"`,
			want:    -3.14,
		},
		{
			name:    "wrong string",
			jsonStr: `"3.f14"`,
			wantErr: true,
		},
		{
			name:    "wrong type",
			jsonStr: `true`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.jsonStr))
			defer jsoniter.ConfigFastest.ReturnIterator(iter)
			val := ReadFloat64(iter)
			if tt.wantErr {
				assert.Error(t, iter.Error)
				return
			}
			assert.NoError(t, iter.Error)
			assert.Equal(t, tt.want, val)
		})
	}
}
