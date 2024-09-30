// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configcompression // import "go.opentelemetry.io/collector/config/configcompression"

import (
	"fmt"

	"github.com/klauspost/compress/zlib"
)

// Type represents a compression method
type Type string
type Level int

type TypeWithLevel struct {
	Type  Type  `mapstructure:"type"`
	Level Level `mapstructure:"level"`
}

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

func (ct *TypeWithLevel) UnmarshalText() (TypeWithLevel, error) {
	typ := ct.Type
	if (typ == TypeGzip && isValidLevel(int(ct.Level))) ||
		(typ == TypeZlib && isValidLevel(int(ct.Level))) ||
		(typ == TypeDeflate && isValidLevel(int(ct.Level))) ||
		typ == TypeSnappy ||
		typ == TypeZstd ||
		typ == TypeLz4 ||
		typ == typeNone ||
		typ == typeEmpty {
		return TypeWithLevel{Type: typ, Level: ct.Level}, nil
	}

	return TypeWithLevel{Type: typ, Level: ct.Level}, fmt.Errorf("unsupported compression type/level %q/%q", typ, ct.Level)
}

func isValidLevel(level int) bool {
	return level == zlib.BestSpeed ||
		level == zlib.BestCompression ||
		level == zlib.DefaultCompression ||
		level == zlib.HuffmanOnly ||
		level == zlib.NoCompression
}
