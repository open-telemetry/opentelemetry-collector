// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

import (
	"fmt"
	"strings"
)

// FieldInterface temporary interface until we generate the proto fields with pdatagen.
// TODO: Remove when no more wrappers needed.
type FieldInterface interface {
	GenTestFailingUnmarshalProtoValues() string

	GenTestEncodingValues() string

	GenPool() string

	GenDelete() string

	GenCopy() string

	GenMarshalJSON() string

	GenUnmarshalJSON() string

	GenSizeProto() string

	GenMarshalProto() string

	GenUnmarshalProto() string

	GenMessageField() string

	GenOneOfMessages() string

	GoType() string

	DefaultValue() string

	TestValue() string

	GetName() string
}

type Field struct {
	Type              Type
	Name              string
	OneOfGroup        string
	OneOfMessageName  string
	MessageName       string
	ParentMessageName string
	ID                uint32
	Repeated          bool
	Nullable          bool
}

func (pf *Field) GetName() string { return pf.Name }

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
	default:
		panic("unsupported field type")
	}
}

func (pf *Field) DefaultValue() string {
	switch pf.Type {
	case TypeInt32, TypeInt64, TypeUint32, TypeUint64, TypeSInt32, TypeSInt64, TypeEnum, TypeFixed32, TypeSFixed32, TypeFloat, TypeFixed64, TypeSFixed64, TypeDouble:
		return pf.GoType() + `(0)`
	case TypeBool:
		return `false`
	case TypeBytes:
		return `nil`
	case TypeString:
		return `""`
	case TypeMessage:
		if pf.Nullable {
			return `&` + pf.MessageName + `{}`
		}
		return pf.MessageName + `{}`
	default:
		panic("unsupported field type")
	}
}

func (pf *Field) TestValue() string {
	if pf.Repeated {
		return pf.MemberGoType() + "{" + pf.DefaultValue() + ", " + pf.rawTestValue() + "}"
	}
	return pf.rawTestValue()
}

func (pf *Field) rawTestValue() string {
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
	case TypeMessage:
		if pf.Nullable {
			return `GenTest` + pf.MessageName + `()`
		}
		return `*GenTest` + pf.MessageName + `()`
	default:
		panic("unsupported field type")
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
		return pf.MessageName
	default:
		panic("unsupported field type")
	}
}

func (pf *Field) MemberGoType() string {
	ptrGoType := func() string {
		if pf.Nullable {
			return "*" + pf.GoType()
		}
		return pf.GoType()
	}
	if pf.Repeated {
		return "[]" + ptrGoType()
	}
	return ptrGoType()
}

func (pf *Field) GenMessageField() string {
	return pf.Name + " " + pf.MemberGoType()
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
		"protoTagSize":      len(protoTag),
		"protoTag":          protoTag,
		"protoFieldID":      pf.ID,
		"jsonTag":           genJSONTag(pf.Name),
		"fieldName":         pf.Name,
		"messageName":       pf.MessageName,
		"parentMessageName": pf.ParentMessageName,
		"oneOfGroup":        pf.OneOfGroup,
		"oneOfMessageName":  pf.OneOfMessageName,
		"repeated":          pf.Repeated,
		"nullable":          pf.Nullable,
		"bitSize":           bitSize,
		"goType":            pf.GoType(),
		"defaultValue":      pf.DefaultValue(),
		"testValue":         pf.TestValue(),
	}
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
