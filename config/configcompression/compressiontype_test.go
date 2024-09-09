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

func TestIsZstd(t *testing.T) {
	tests := []struct {
		name     string
		input    Type
		expected bool
	}{
		{
			name:     "ValidZstd",
			input:    TypeZstd,
			expected: true,
		},
		{
			name:     "InvalidZstd",
			input:    TypeGzip,
			expected: false,
		},
		{
			name:     "ValidZstdLevel",
			input:    "zstd/11",
			expected: true,
		},
		{
			name:     "ValidZstdLevel",
			input:    "zstd/One",
			expected: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input.IsZstd())
		})
	}
}

func TestIsGzip(t *testing.T) {
	tests := []struct {
		name     string
		input    Type
		expected bool
	}{
		{
			name:     "ValidGzip",
			input:    TypeGzip,
			expected: true,
		},
		{
			name:     "InvalidGzip",
			input:    TypeZlib,
			expected: false,
		},
		{
			name:     "ValidZlibCompressionLevel",
			input:    "gzip/1",
			expected: true,
		},
		{
			name:     "InvalidZlibCompressionLevel",
			input:    "gzip/10",
			expected: false,
		},
		{
			name:     "InvalidZlibCompressionLevel",
			input:    "gzip/one", // Uses the default compression level
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input.IsGzip())
		})
	}
}

func TestIsZlib(t *testing.T) {
	tests := []struct {
		name     string
		input    Type
		expected bool
		err      bool
	}{
		{
			name:     "ValidZlib",
			input:    TypeZlib,
			expected: true,
		},
		{
			name:     "InvalidZlib",
			input:    TypeGzip,
			expected: false,
		},
		{
			name:     "ValidZlibCompressionLevel",
			input:    "zlib/1",
			expected: true,
		},
		{
			name:     "InvalidZlibCompressionLevel",
			input:    "zlib/10",
			expected: false,
		},
		{
			name:     "InvalidZlibCompressionLevel",
			input:    "zlib/one", // Uses the default compression level
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input.IsZlib())
		})
	}
}

func TestIsSnappy(t *testing.T) {
	tests := []struct {
		name     string
		input    Type
		expected bool
		err      bool
	}{
		{
			name:     "ValidSnappy",
			input:    "snappy",
			expected: true,
		},
		{
			name:     "InvliaSnappy",
			input:    "snappy/1", // Uses the default compression level
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input.IsSnappy())
		})
	}
}
