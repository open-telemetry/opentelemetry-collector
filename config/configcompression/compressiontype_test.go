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
		isCompressed    bool
	}{
		{
			name:            "ValidGzip",
			compressionName: []byte("gzip"),
			shouldError:     false,
			isCompressed:    true,
		},
		{
			name:            "ValidZlib",
			compressionName: []byte("zlib"),
			shouldError:     false,
			isCompressed:    true,
		},
		{
			name:            "ValidDeflate",
			compressionName: []byte("deflate"),
			shouldError:     false,
			isCompressed:    true,
		},
		{
			name:            "ValidSnappy",
			compressionName: []byte("snappy"),
			shouldError:     false,
			isCompressed:    true,
		},
		{
			name:            "ValidZstd",
			compressionName: []byte("zstd"),
			shouldError:     false,
			isCompressed:    true,
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
			temp := typeNone
			err := temp.UnmarshalText(tt.compressionName)
			if tt.shouldError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			ct := Type(tt.compressionName)
			assert.Equal(t, temp, ct)
			assert.Equal(t, tt.isCompressed, ct.IsCompressed())
		})
	}
}
