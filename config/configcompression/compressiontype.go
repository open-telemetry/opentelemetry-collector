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

func IsCompressed(compressionType CompressionType) bool {
	return compressionType != empty && compressionType != none
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
