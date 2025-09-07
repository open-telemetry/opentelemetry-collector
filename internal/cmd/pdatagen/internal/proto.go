// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"fmt"
	"strings"
)

// WireType represents the proto wire type.
type WireType uint32

const (
	WireTypeVarint     WireType = 0
	WireTypeI64        WireType = 1
	WireTypeLen        WireType = 2
	WireTypeStartGroup WireType = 3
	WireTypeEndGroup   WireType = 4
	WireTypeI32        WireType = 5
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

	ProtoTypeMessage
)

func (pt ProtoType) goType(messageName string) string {
	switch pt {
	case ProtoTypeDouble:
		return "float64"
	case ProtoTypeFloat:
		return "float32"
	case ProtoTypeInt32, ProtoTypeSInt32, ProtoTypeSFixed32:
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
	case ProtoTypeMessage, ProtoTypeEnum:
		return messageName
	default:
		panic("unreachable")
	}
}

func (pt ProtoType) defaultValue(messageName string) string {
	switch pt {
	case ProtoTypeInt32, ProtoTypeInt64, ProtoTypeUint32, ProtoTypeUint64, ProtoTypeSInt32, ProtoTypeSInt64, ProtoTypeEnum, ProtoTypeFixed32, ProtoTypeSFixed32, ProtoTypeFloat, ProtoTypeFixed64, ProtoTypeSFixed64, ProtoTypeDouble:
		return pt.goType(messageName) + `(0)`
	case ProtoTypeBool:
		return `false`
	case ProtoTypeBytes:
		return `[]byte{}`
	case ProtoTypeString:
		return `""`
	case ProtoTypeMessage:
		return messageName + `{}`
	default:
		panic("unreachable")
	}
}

func (pt ProtoType) testValue(fieldName string) string {
	switch pt {
	case ProtoTypeInt32, ProtoTypeInt64, ProtoTypeUint32, ProtoTypeUint64, ProtoTypeSInt32, ProtoTypeSInt64, ProtoTypeEnum, ProtoTypeFixed32, ProtoTypeSFixed32, ProtoTypeFixed64, ProtoTypeSFixed64:
		return pt.goType("") + "(13)"
	case ProtoTypeFloat, ProtoTypeDouble:
		return pt.goType("") + "(3.1415926)"
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

type ProtoField struct {
	Type        ProtoType
	Name        string
	MessageName string
	ID          uint32
	Repeated    bool
	Nullable    bool
}

func (pf *ProtoField) wireType() WireType {
	switch pf.Type {
	case ProtoTypeInt32, ProtoTypeInt64, ProtoTypeUint32, ProtoTypeUint64, ProtoTypeSInt32, ProtoTypeSInt64, ProtoTypeBool, ProtoTypeEnum:
		// In proto3, repeated scalar types are packed; hence they use Length-Delimited (Wire Type 2).
		if pf.Repeated {
			return WireTypeLen
		}
		return WireTypeVarint
	case ProtoTypeFixed32, ProtoTypeSFixed32, ProtoTypeFloat:
		// In proto3, repeated scalar types are packed; hence they use Length-Delimited (Wire Type 2).
		if pf.Repeated {
			return WireTypeLen
		}
		return WireTypeI32
	case ProtoTypeFixed64, ProtoTypeSFixed64, ProtoTypeDouble:
		// In proto3, repeated scalar types are packed; hence they use Length-Delimited (Wire Type 2).
		if pf.Repeated {
			return WireTypeLen
		}
		return WireTypeI64
	case ProtoTypeBytes, ProtoTypeMessage, ProtoTypeString:
		return WireTypeLen
	}
	panic("unreachable")
}

// genProtoKey encodes the field key, and returns it in the reverse order.
func genProtoKey(fieldNumber uint32, wt WireType) []string {
	x := fieldNumber<<3 | uint32(wt)
	i := 0
	keybuf := make([]byte, 0)
	for i = 0; x > 127; i++ {
		//nolint:gosec
		keybuf = append(keybuf, 0x80|uint8(x&0x7F))
		x >>= 7
	}
	keybuf = append(keybuf, uint8(x))
	ret := make([]string, 0, len(keybuf))
	for i = len(keybuf) - 1; i >= 0; i-- {
		ret = append(ret, fmt.Sprintf("%#v", keybuf[i]))
	}
	return ret
}
