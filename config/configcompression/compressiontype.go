// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configcompression // import "go.opentelemetry.io/collector/config/configcompression"

import (
	"compress/zlib"
	"fmt"
)

// Type represents a compression method
type Type string

type Level int

type CompressionParams struct {
	Level Level `mapstructure:"level"`
	// prevent unkeyed literal initialization
	_ struct{}
}

const (
	TypeGzip                Type = "gzip"
	TypeZlib                Type = "zlib"
	TypeDeflate             Type = "deflate"
	TypeSnappy              Type = "snappy"
	TypeSnappyFramed        Type = "x-snappy-framed"
	TypeZstd                Type = "zstd"
	TypeLz4                 Type = "lz4"
	typeNone                Type = "none"
	typeEmpty               Type = ""
	DefaultCompressionLevel      = zlib.DefaultCompression
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
		typ == TypeSnappyFramed ||
		typ == TypeZstd ||
		typ == TypeLz4 ||
		typ == typeNone ||
		typ == typeEmpty {
		*ct = typ
		return nil
	}
	return fmt.Errorf("unsupported compression type %q", typ)
}

func (ct *Type) ValidateParams(p CompressionParams) error {
	switch *ct {
	case TypeGzip, TypeZlib, TypeDeflate:
		if p.Level == zlib.DefaultCompression ||
			p.Level == zlib.HuffmanOnly ||
			p.Level == zlib.NoCompression ||
			(p.Level >= zlib.BestSpeed && p.Level <= zlib.BestCompression) {
			return nil
		}
	case TypeZstd:
		// Supports arbitrary levels: zstd will map any given
		// level to the nearest internally supported level.
		return nil
	}
	if p.Level != 0 {
		return fmt.Errorf("unsupported parameters {Level:%+v} for compression type %q", p.Level, *ct)
	}
	return nil
}
