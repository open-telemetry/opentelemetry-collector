// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configcompression // import "go.opentelemetry.io/collector/config/configcompression"

import "fmt"

// CompressionType represents a compression method
// Deprecated [0.94.0]: use Type instead.
type CompressionType = Type

// Type represents a compression method
type Type string

const (
	TypeGzip    Type = "gzip"
	TypeZlib    Type = "zlib"
	TypeDeflate Type = "deflate"
	TypeSnappy  Type = "snappy"
	TypeZstd    Type = "zstd"
	typeNone    Type = "none"
	typeEmpty   Type = ""

	// Gzip
	// Deprecated [0.94.0]: use TypeGzip instead.
	Gzip CompressionType = "gzip"

	// Zlib
	// Deprecated [0.94.0]: use TypeZlib instead.
	Zlib CompressionType = "zlib"

	// Deflate
	// Deprecated [0.94.0]: use TypeDeflate instead.
	Deflate CompressionType = "deflate"

	// Snappy
	// Deprecated [0.94.0]: use TypeSnappy instead.
	Snappy CompressionType = "snappy"

	// Zstd
	// Deprecated [0.94.0]: use TypeZstd instead.
	Zstd CompressionType = "zstd"
)

// IsCompressed returns false if CompressionType is nil, none, or empty. Otherwise it returns true.
//
// Deprecated: [0.94.0] use member function CompressionType.IsCompressed instead
func IsCompressed(compressionType CompressionType) bool {
	return compressionType.IsCompressed()
}

// IsCompressed returns false if CompressionType is nil, none, or empty.
// Otherwise, returns true.
func (ct *Type) IsCompressed() bool {
	return *ct != typeEmpty && *ct != typeNone
}

func (ct *Type) UnmarshalText(in []byte) error {
	switch typ := Type(in); typ {
	case TypeGzip,
		TypeZlib,
		TypeDeflate,
		TypeSnappy,
		TypeZstd,
		typeNone,
		typeEmpty:
		*ct = typ
		return nil
	default:
		return fmt.Errorf("unsupported compression type %q", typ)
	}
}
