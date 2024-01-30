// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configcompression // import "go.opentelemetry.io/collector/config/configcompression"

import "fmt"

type CompressionType string

const (
	Gzip    CompressionType = "gzip"
	Zlib    CompressionType = "zlib"
	Deflate CompressionType = "deflate"
	Snappy  CompressionType = "snappy"
	Zstd    CompressionType = "zstd"
	none    CompressionType = "none"
	empty   CompressionType = ""
)

// IsCompressed Deprecated [0.94.0] use member function CompressionType.IsCompressed instead
func IsCompressed(compressionType CompressionType) bool {
	return compressionType != empty && compressionType != none
}

// IsCompressed returns false if CompressionType is nil, none, or empty.
// Otherwise, returns true.
func (ct *CompressionType) IsCompressed() bool {
	return ct != nil && *ct != empty && *ct != none
}

func (ct *CompressionType) UnmarshalText(in []byte) error {
	switch typ := CompressionType(in); typ {
	case Gzip,
		Zlib,
		Deflate,
		Snappy,
		Zstd,
		none,
		empty:
		*ct = typ
		return nil
	default:
		return fmt.Errorf("unsupported compression type %q", typ)
	}
}
