// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"fmt"
	"strings"
)

type Field struct {
	Type                 Type
	Name                 string
	OneOfGroup           string
	OneOfMessageFullName string
	MessageFullName      string
	ID                   uint32
	Repeated             bool
	Nullable             bool
}

func (pf *Field) wireType() WireType {
	switch pf.Type {
	case TypeInt32, TypeInt64, TypeUint32, TypeUint64, TypeSInt32, TypeSInt64, TypeBool, TypeEnum:
		// In proto3, repeated scalar types are packed; hence they use Length-Delimited (Wire Type 2).
		if pf.Repeated {
			return WireTypeLen
		}
		return WireTypeVarint
	case TypeFixed32, TypeSFixed32, TypeFloat:
		// In proto3, repeated scalar types are packed; hence they use Length-Delimited (Wire Type 2).
		if pf.Repeated {
			return WireTypeLen
		}
		return WireTypeI32
	case TypeFixed64, TypeSFixed64, TypeDouble:
		// In proto3, repeated scalar types are packed; hence they use Length-Delimited (Wire Type 2).
		if pf.Repeated {
			return WireTypeLen
		}
		return WireTypeI64
	case TypeBytes, TypeMessage, TypeString:
		return WireTypeLen
	}
	panic("unreachable")
}

func (pf *Field) DefaultValue() string {
	switch pf.Type {
	case TypeInt32, TypeInt64, TypeUint32, TypeUint64, TypeSInt32, TypeSInt64, TypeEnum, TypeFixed32, TypeSFixed32, TypeFloat, TypeFixed64, TypeSFixed64, TypeDouble:
		return pf.GoType() + `(0)`
	case TypeBool:
		return `false`
	case TypeBytes:
		return `[]byte{}`
	case TypeString:
		return `""`
	case TypeMessage:
		return pf.MessageFullName + `{}`
	default:
		panic("unreachable")
	}
}

func (pf *Field) TestValue() string {
	switch pf.Type {
	case TypeInt32, TypeInt64, TypeUint32, TypeUint64, TypeSInt32, TypeSInt64, TypeEnum, TypeFixed32, TypeSFixed32, TypeFixed64, TypeSFixed64:
		return pf.GoType() + "(13)"
	case TypeFloat, TypeDouble:
		return pf.GoType() + "(3.1415926)"
	case TypeBool:
		return `true`
	case TypeBytes:
		return `[]byte{1, 2, 3}`
	case TypeString:
		return `"test_` + strings.ToLower(pf.Name) + `"`
	default:
		panic("unreachable")
	}
}

func (pf *Field) GoType() string {
	switch pf.Type {
	case TypeDouble:
		return "float64"
	case TypeFloat:
		return "float32"
	case TypeInt32, TypeSInt32, TypeSFixed32:
		return "int32"
	case TypeInt64, TypeSInt64, TypeSFixed64:
		return "int64"
	case TypeUint32, TypeFixed32:
		return "uint32"
	case TypeUint64, TypeFixed64:
		return "uint64"
	case TypeBool:
		return "bool"
	case TypeString:
		return "string"
	case TypeBytes:
		return "[]byte"
	case TypeMessage, TypeEnum:
		return pf.MessageFullName
	default:
		panic("unreachable")
	}
}

func (pf *Field) getTemplateFields() map[string]any {
	bitSize := 0
	switch pf.Type {
	case TypeFixed64, TypeSFixed64, TypeInt64, TypeUint64, TypeSInt64, TypeDouble:
		bitSize = 64
	case TypeFixed32, TypeSFixed32, TypeInt32, TypeUint32, TypeSInt32, TypeFloat, TypeEnum:
		bitSize = 32
	}

	protoTag := genProtoTag(pf.ID, pf.wireType())
	return map[string]any{
		"protoTagSize":         len(protoTag),
		"protoTag":             protoTag,
		"protoFieldID":         pf.ID,
		"jsonTag":              genJSONTag(pf.Name),
		"fieldName":            pf.Name,
		"origName":             pf.messageName(),
		"oneOfGroup":           pf.OneOfGroup,
		"oneOfMessageFullName": pf.OneOfMessageFullName,
		"repeated":             pf.Repeated,
		"nullable":             pf.Nullable,
		"bitSize":              bitSize,
		"goType":               pf.GoType(),
		"defaultValue":         pf.DefaultValue(),
	}
}

func (pf *Field) messageName() string {
	return ExtractNameFromFull(pf.MessageFullName)
}

func genJSONTag(fieldName string) string {
	// Extract last word because for Enums we use the full name.
	return lowerFirst(ExtractNameFromFull(fieldName))
}

// genProtoTag encodes the field key, and returns it in the reverse order.
func genProtoTag(fieldNumber uint32, wt WireType) []string {
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

func ExtractNameFromFull(fullName string) string {
	// Extract last word because for Enums we use the full name.
	lastSpaceIndex := strings.LastIndex(fullName, ".")
	if lastSpaceIndex != -1 {
		return fullName[lastSpaceIndex+1:]
	}
	return fullName
}

func lowerFirst(s string) string {
	return strings.ToLower(s[0:1]) + s[1:]
}
