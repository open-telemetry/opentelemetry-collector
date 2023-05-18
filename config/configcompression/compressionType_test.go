// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
