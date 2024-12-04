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

type CompressionConfig struct {
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

var typ Type

// IsCompressed returns false if CompressionType is nil, none, or empty.
// Otherwise, returns true.
func (ct *Type) IsCompressed() bool {
	return *ct != typeEmpty && *ct != typeNone
}

func (ct *Type) UnmarshalText(in []byte) error {
	typ = Type(in)
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

func (cc *CompressionConfig) Validate() error {
	if (typ == TypeGzip && isValidLevel(int(cc.Level))) ||
		(typ == TypeZlib && isValidLevel(int(cc.Level))) ||
		(typ == TypeDeflate && isValidLevel(int(cc.Level))) ||
		typ == TypeSnappy ||
		typ == TypeLz4 ||
		typ == TypeZstd ||
		typ == typeNone ||
		typ == typeEmpty {
		return nil
	}

	return fmt.Errorf("unsupported compression type and level %s - %d", typ, cc.Level)
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
