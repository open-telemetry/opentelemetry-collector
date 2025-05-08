// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configcompression

import (
	"compress/zlib"
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
			name:            "ValidSnappyFramed",
			compressionName: []byte("x-snappy-framed"),
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

func TestValidateParams(t *testing.T) {
	tests := []struct {
		name             string
		compressionName  []byte
		compressionLevel Level
		shouldError      bool
	}{
		{
			name:             "ValidGzip",
			compressionName:  []byte("gzip"),
			compressionLevel: zlib.DefaultCompression,
			shouldError:      false,
		},
		{
			name:             "InvalidGzip",
			compressionName:  []byte("gzip"),
			compressionLevel: 99,
			shouldError:      true,
		},
		{
			name:             "ValidZlib",
			compressionName:  []byte("zlib"),
			compressionLevel: zlib.DefaultCompression,
			shouldError:      false,
		},
		{
			name:             "ValidDeflate",
			compressionName:  []byte("deflate"),
			compressionLevel: zlib.DefaultCompression,
			shouldError:      false,
		},
		{
			name:            "ValidSnappy",
			compressionName: []byte("snappy"),
			shouldError:     false,
		},
		{
			name:             "InvalidSnappy",
			compressionName:  []byte("snappy"),
			compressionLevel: 1,
			shouldError:      true,
		},
		{
			name:            "ValidSnappyFramed",
			compressionName: []byte("x-snappy-framed"),
			shouldError:     false,
		},
		{
			name:             "InvalidSnappyFramed",
			compressionName:  []byte("x-snappy-framed"),
			compressionLevel: 1,
			shouldError:      true,
		},
		{
			name:             "ValidZstd",
			compressionName:  []byte("zstd"),
			compressionLevel: zlib.DefaultCompression,
			shouldError:      false,
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
			shouldError:     false,
		},
		{
			name:             "InvalidLz4",
			compressionName:  []byte("lz4"),
			compressionLevel: 1,
			shouldError:      true,
		},
		{
			name:             "Invalid",
			compressionName:  []byte("ggip"),
			compressionLevel: zlib.DefaultCompression,
			shouldError:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressionParams := CompressionParams{Level: tt.compressionLevel}
			temp := Type(tt.compressionName)
			err := temp.ValidateParams(compressionParams)
			if tt.shouldError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
