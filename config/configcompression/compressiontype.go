// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configcompression // import "go.opentelemetry.io/collector/config/configcompression"

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/klauspost/compress/zlib"
)

// Type represents a compression method
type Type string

const (
	TypeGzip    Type = "gzip"
	TypeZlib    Type = "zlib"
	TypeDeflate Type = "deflate"
	TypeSnappy  Type = "snappy"
	TypeZstd    Type = "zstd"
	TypeLz4     Type = "lz4"
	typeNone    Type = "none"
	typeEmpty   Type = ""
)

// IsCompressed returns false if CompressionType is nil, none, or empty.
// Otherwise, returns true.
func (ct *Type) IsCompressed() bool {
	return *ct != typeEmpty && *ct != typeNone
}

func (ct *Type) UnmarshalText(in []byte) error {
	typ := Type(in)
	if typ == TypeGzip ||
		typ == TypeZlib ||
		typ == TypeDeflate ||
		typ == TypeSnappy ||
		typ == TypeZstd ||
		typ == TypeLz4 ||
		typ == typeNone ||
		typ == typeEmpty {
		*ct = typ
		return nil
	}
	return fmt.Errorf("unsupported compression type %q", typ)

}

// IsZstd returns true if the compression type is zstd.
func (ct *Type) IsZstd() bool {
	parts := strings.Split(string(*ct), "/")
	return parts[0] == string(TypeZstd)
}

// IsGzip returns true if the compression type is gzip and the specified compression level is valid.
func (ct *Type) IsGzip() bool {
	parts := strings.Split(string(*ct), "/")
	if parts[0] == string(TypeGzip) {
		if len(parts) > 1 {
			levelStr, err := strconv.Atoi(parts[1])
			if err != nil {
				return false
			}
			if levelStr == zlib.BestSpeed ||
				levelStr == zlib.BestCompression ||
				levelStr == zlib.DefaultCompression ||
				levelStr == zlib.HuffmanOnly ||
				levelStr == zlib.NoCompression {
				return true
			}
		}
		return true
	}
	return false
}

// IsZlib returns true if the compression type is zlib and the specified compression level is valid.
func (ct *Type) IsZlib() bool {
	parts := strings.Split(string(*ct), "/")
	if parts[0] == string(TypeZlib) || parts[0] == string(TypeDeflate) {
		if len(parts) > 1 {
			levelStr, err := strconv.Atoi(parts[1])
			if err != nil {
				return false
			}
			if levelStr == zlib.BestSpeed ||
				levelStr == zlib.BestCompression ||
				levelStr == zlib.DefaultCompression ||
				levelStr == zlib.HuffmanOnly ||
				levelStr == zlib.NoCompression {
				return true
			}
		}
		return true
	}
	return false
}
