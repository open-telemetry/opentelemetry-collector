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
	TypeGzip    Type  = "gzip"
	TypeZlib    Type  = "zlib"
	TypeDeflate Type  = "deflate"
	TypeSnappy  Type  = "snappy"
	TypeZstd    Type  = "zstd"
	TypeLz4     Type  = "lz4"
	typeNone    Type  = "none"
	typeEmpty   Type  = ""
	LevelNone   Level = 0
)

// IsCompressed returns false if CompressionType is nil, none, or empty.
// Otherwise, returns true.
func (ct *Type) IsCompressed() bool {
	return *ct != typeEmpty && *ct != typeNone
}

func (ct *TypeWithLevel) UnmarshalText(in []byte) error {
	typ := Type(in)
	if typ == TypeGzip ||
		typ == TypeZlib ||
		typ == TypeDeflate ||
		typ == TypeSnappy ||
		typ == TypeZstd ||
		typ == TypeLz4 ||
		typ == typeNone ||
		typ == typeEmpty {
		*&ct.Type = typ
		return nil
	}
	return fmt.Errorf("unsupported compression type %q", typ)
}

func (ct *TypeWithLevel) Validate() error {
	if (ct.Type == TypeGzip && isValidLevel(int(ct.Level))) ||
		(ct.Type == TypeZlib && isValidLevel(int(ct.Level))) ||
		(ct.Type == TypeDeflate && isValidLevel(int(ct.Level))) ||
		ct.Type == TypeSnappy ||
		ct.Type == TypeLz4 ||
		ct.Type == TypeZstd ||
		ct.Type == typeNone ||
		ct.Type == typeEmpty {
		return nil
	}

	return fmt.Errorf("unsupported compression type and level %s - %d", ct.Type, ct.Level)
}

// Checks the validity of zlib/gzip/flate compression levels
func isValidLevel(level int) bool {
	return level == zlib.DefaultCompression ||
		level == int(LevelNone) ||
		level == zlib.HuffmanOnly ||
		level == zlib.NoCompression ||
		level == zlib.BestSpeed ||
		(level >= zlib.BestSpeed && level <= zlib.BestCompression)
}
