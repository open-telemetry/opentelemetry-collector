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
type Level int

type TypeWithLevel struct {
	Type  Type
	Level Level
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

func (ct *TypeWithLevel) UnmarshalText(in []byte) error {
	var compressionTyp Type
	var level int
	var err error
	parts := strings.Split(string(in), "/")
	compressionTyp = Type(parts[0])
	level = zlib.DefaultCompression
	if len(parts) == 2 {
		level, err = strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("invalid compression level: %q", parts[1])
		}
		if compressionTyp == TypeSnappy ||
			compressionTyp == typeNone ||
			compressionTyp == typeEmpty {
			return fmt.Errorf("compression level is not supported for %q", compressionTyp)
		}
	}
	ct.Level = Level(level)
	if (compressionTyp == TypeGzip && isValidLevel(level)) ||
		(compressionTyp == TypeZlib && isValidLevel(level)) ||
		(compressionTyp == TypeDeflate && isValidLevel(level)) ||
		compressionTyp == TypeSnappy ||
		compressionTyp == TypeZstd ||
		compressionTyp == typeNone ||
		compressionTyp == typeEmpty {
		ct.Level = Level(level)
		ct.Type = compressionTyp
		return nil
	}

	return fmt.Errorf("unsupported compression type/level %s/%d", compressionTyp, ct.Level)

}

func isValidLevel(level int) bool {
	return level == zlib.BestSpeed ||
		level == zlib.BestCompression ||
		level == zlib.DefaultCompression ||
		level == zlib.HuffmanOnly ||
		level == zlib.NoCompression
}
