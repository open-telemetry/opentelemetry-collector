// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
