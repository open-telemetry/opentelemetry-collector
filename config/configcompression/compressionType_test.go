// Copyright  The OpenTelemetry Authors
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

package configcompression

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalText(t *testing.T) {
	tests := []struct {
		name            string
		compressionName []byte
		shouldError     bool
	}{
		{
			name:            "ValidGzip",
			compressionName: []byte("gzip"),
			shouldError:     false,
		},
		{
			name:            "ValidZlib",
			compressionName: []byte("zlib"),
			shouldError:     false,
		},
		{
			name:            "ValidDeflate",
			compressionName: []byte("deflate"),
			shouldError:     false,
		},
		{
			name:            "ValidSnappy",
			compressionName: []byte("snappy"),
			shouldError:     false,
		},
		{
			name:            "ValidZstd",
			compressionName: []byte("zstd"),
			shouldError:     false,
		},
		{
			name:            "ValidEmpty",
			compressionName: []byte(""),
			shouldError:     false,
		},
		{
			name:            "ValidNone",
			compressionName: []byte("none"),
			shouldError:     false,
		},
		{
			name:            "Invalid",
			compressionName: []byte("ggip"),
			shouldError:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			temp := none
			err := temp.UnmarshalText(tt.compressionName)
			if tt.shouldError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, temp, CompressionType(tt.compressionName))
		})
	}
}
