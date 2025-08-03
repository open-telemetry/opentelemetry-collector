// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"strings"
)

type ProtoType int32

const (
	ProtoTypeDouble ProtoType = iota
	ProtoTypeFloat

	ProtoTypeInt32
	ProtoTypeInt64
	ProtoTypeUint32
	ProtoTypeUint64

	ProtoTypeSInt32
	ProtoTypeSInt64

	ProtoTypeFixed32
	ProtoTypeFixed64
	ProtoTypeSFixed32
	ProtoTypeSFixed64

	ProtoTypeBool
	ProtoTypeEnum

	ProtoTypeString
	ProtoTypeBytes
)

func (pt ProtoType) goType() string {
	switch pt {
	case ProtoTypeDouble:
		return "float64"
	case ProtoTypeFloat:
		return "float32"
	case ProtoTypeInt32, ProtoTypeSInt32, ProtoTypeSFixed32, ProtoTypeEnum:
		return "int32"
	case ProtoTypeInt64, ProtoTypeSInt64, ProtoTypeSFixed64:
		return "int64"
	case ProtoTypeUint32, ProtoTypeFixed32:
		return "uint32"
	case ProtoTypeUint64, ProtoTypeFixed64:
		return "uint64"
	case ProtoTypeBool:
		return "bool"
	case ProtoTypeString:
		return "string"
	case ProtoTypeBytes:
		return "[]byte"
	default:
		panic("unreachable")
	}
}

func (pt ProtoType) defaultValue() string {
	switch pt {
	case ProtoTypeInt32, ProtoTypeInt64, ProtoTypeUint32, ProtoTypeUint64, ProtoTypeSInt32, ProtoTypeSInt64, ProtoTypeEnum, ProtoTypeFixed32, ProtoTypeSFixed32, ProtoTypeFloat, ProtoTypeFixed64, ProtoTypeSFixed64, ProtoTypeDouble:
		return pt.goType() + `(0)`
	case ProtoTypeBool:
		return `false`
	case ProtoTypeBytes:
		return `[]byte{}`
	case ProtoTypeString:
		return `""`
	default:
		panic("unreachable")
	}
}

func (pt ProtoType) testValue(fieldName string) string {
	switch pt {
	case ProtoTypeInt32, ProtoTypeInt64, ProtoTypeUint32, ProtoTypeUint64, ProtoTypeSInt32, ProtoTypeSInt64, ProtoTypeEnum, ProtoTypeFixed32, ProtoTypeSFixed32, ProtoTypeFixed64, ProtoTypeSFixed64:
		return pt.goType() + "(13)"
	case ProtoTypeFloat, ProtoTypeDouble:
		return pt.goType() + "(3.1415926)"
	case ProtoTypeBool:
		return `true`
	case ProtoTypeBytes:
		return `[]byte{1, 2, 3}`
	case ProtoTypeString:
		return `"test_` + strings.ToLower(fieldName) + `"`
	default:
		panic("unreachable")
	}
}
