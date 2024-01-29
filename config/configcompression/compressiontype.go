// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configcompression // import "go.opentelemetry.io/collector/config/configcompression"

import "fmt"

type CompressionType string

const (
	CompressionTypeGzip    CompressionType = "gzip"
	CompressionTypeZlib    CompressionType = "zlib"
	CompressionTypeDeflate CompressionType = "deflate"
	CompressionTypeSnappy  CompressionType = "snappy"
	CompressionTypeZstd    CompressionType = "zstd"
	compressionTypeNone    CompressionType = "none"
	compressionTypeEmpty   CompressionType = ""
)

func IsCompressed(compressionType CompressionType) bool {
	return compressionType != compressionTypeEmpty && compressionType != compressionTypeNone
}

func (ct *CompressionType) UnmarshalText(in []byte) error {
	switch typ := CompressionType(in); typ {
	case CompressionTypeGzip,
		CompressionTypeZlib,
		CompressionTypeDeflate,
		CompressionTypeSnappy,
		CompressionTypeZstd,
		compressionTypeNone,
		compressionTypeEmpty:
		*ct = typ
		return nil
	default:
		return fmt.Errorf("unsupported compression type %q", typ)
	}
}
