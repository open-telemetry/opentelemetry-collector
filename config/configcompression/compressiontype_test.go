// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configcompression

import (
	"strings"
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
			name:            "ValidZstdLevel",
			compressionName: []byte("zstd/11"),
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
			name:            "ValidLz4",
			compressionName: []byte("lz4"),
			isCompressed:    true,
			shouldError:     false,
		},
		{
			name:            "Invalid",
			compressionName: []byte("ggip"),
			shouldError:     true,
		},
		{
			name:            "InvalidSnappy",
			compressionName: []byte("snappy/1"),
			shouldError:     true,
		},
		{
			name:            "InvalidNone",
			compressionName: []byte("none/1"),
			shouldError:     true,
		},
		{
			name:            "InvalidGzip",
			compressionName: []byte("gzip/10"),
			shouldError:     true,
			isCompressed:    true,
		},
		{
			name:            "InvalidZlib",
			compressionName: []byte("zlib/10"),
			shouldError:     true,
			isCompressed:    true,
		},
		{
			name:            "InvalidZstdLevel",
			compressionName: []byte("zstd/ten"),
			shouldError:     true,
			isCompressed:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var temp TypeWithLevel
			err := temp.UnmarshalText(tt.compressionName)
			if tt.shouldError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			ct := Type(strings.Split(string(tt.compressionName), "/")[0])
			assert.Equal(t, temp.Type, ct)
			assert.Equal(t, tt.isCompressed, ct.IsCompressed())
		})
	}
}
