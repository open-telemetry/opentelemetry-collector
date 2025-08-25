// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	"strings"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const typedAccessorsTemplate = `// {{ .fieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .packageName }}{{ .returnType }} {
	return {{ .packageName }}{{ .returnType }}(ms.orig.{{ .originFieldName }})
}

// Set{{ .fieldName }} replaces the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) Set{{ .fieldName }}(v {{ .packageName }}{{ .returnType }}) {
	ms.state.AssertMutable()
	ms.orig.{{ .originFieldName }} = {{ .rawType }}(v)
}`

const typedAccessorsTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	assert.Equal(t, {{ .packageName }}{{ .returnType }}({{ .defaultVal }}), ms.{{ .fieldName }}())
	testVal{{ .fieldName }} := {{ .packageName }}{{ .returnType }}({{ .testValue }})
	ms.Set{{ .fieldName }}(testVal{{ .fieldName }})
	assert.Equal(t, testVal{{ .fieldName }}, ms.{{ .fieldName }}())
}`

const typedSetTestTemplate = `orig.{{ .originFieldName }} = {{ .testValue }}`

const typedCopyOrigTemplate = `dest.{{ .originFieldName }} = src.{{ .originFieldName }}`

// TypedField is a field that has defined a custom type (e.g. "type Timestamp uint64")
type TypedField struct {
	fieldName       string
	originFieldName string
	protoID         uint32
	returnType      *TypedType
}

type TypedType struct {
	structName  string
	packageName string

	protoType   proto.Type
	messageName string

	defaultVal string
	testVal    string
}

func (ptf *TypedField) GenerateAccessors(ms *messageStruct) string {
	t := template.Parse("typedAccessorsTemplate", []byte(typedAccessorsTemplate))
	return template.Execute(t, ptf.templateFields(ms))
}

func (ptf *TypedField) GenerateAccessorsTest(ms *messageStruct) string {
	t := template.Parse("typedAccessorsTestTemplate", []byte(typedAccessorsTestTemplate))
	return template.Execute(t, ptf.templateFields(ms))
}

func (ptf *TypedField) GenerateTestValue(ms *messageStruct) string {
	t := template.Parse("typedSetTestTemplate", []byte(typedSetTestTemplate))
	return template.Execute(t, ptf.templateFields(ms))
}

func (ptf *TypedField) GenerateTestFailingUnmarshalProtoValues(*messageStruct) string {
	return ptf.toProtoField().GenTestFailingUnmarshalProtoValues()
}

func (ptf *TypedField) GenerateTestEncodingValues(*messageStruct) string {
	return ptf.toProtoField().GenTestEncodingValues()
}

func (ptf *TypedField) GeneratePoolOrig(*messageStruct) string {
	return ""
}

func (ptf *TypedField) GenerateDeleteOrig(*messageStruct) string {
	return ptf.toProtoField().GenDeleteOrig()
}

func (ptf *TypedField) GenerateCopyOrig(ms *messageStruct) string {
	t := template.Parse("typedCopyOrigTemplate", []byte(typedCopyOrigTemplate))
	return template.Execute(t, ptf.templateFields(ms))
}

func (ptf *TypedField) GenerateMarshalJSON(*messageStruct) string {
	if strings.HasPrefix(ptf.returnType.messageName, "data.") {
		return "if orig." + ptf.getOriginFieldName() + " != " + ptf.returnType.defaultVal + "{\n" + ptf.toProtoField().GenMarshalJSON() + "\n}"
	}
	return ptf.toProtoField().GenMarshalJSON()
}

func (ptf *TypedField) GenerateUnmarshalJSON(*messageStruct) string {
	return ptf.toProtoField().GenUnmarshalJSON()
}

func (ptf *TypedField) GenerateSizeProto(*messageStruct) string {
	return ptf.toProtoField().GenSizeProto()
}

func (ptf *TypedField) GenerateMarshalProto(*messageStruct) string {
	return ptf.toProtoField().GenMarshalProto()
}

func (ptf *TypedField) GenerateUnmarshalProto(*messageStruct) string {
	return ptf.toProtoField().GenUnmarshalProto()
}

func (ptf *TypedField) toProtoField() *proto.Field {
	return &proto.Field{
		Type:            ptf.returnType.protoType,
		ID:              ptf.protoID,
		Name:            ptf.getOriginFieldName(),
		MessageFullName: ptf.returnType.messageName,
	}
}

func (ptf *TypedField) templateFields(ms *messageStruct) map[string]any {
	pf := ptf.toProtoField()
	return map[string]any{
		"structName": ms.getName(),
		"defaultVal": ptf.returnType.defaultVal,
		"packageName": func() string {
			if ptf.returnType.packageName != ms.packageName {
				return ptf.returnType.packageName + "."
			}
			return ""
		}(),
		"hasWrapper":      usedByOtherDataTypes(ptf.returnType.packageName),
		"returnType":      ptf.returnType.structName,
		"fieldName":       ptf.fieldName,
		"lowerFieldName":  strings.ToLower(ptf.fieldName),
		"testValue":       ptf.returnType.testVal,
		"rawType":         pf.GoType(),
		"originFieldName": ptf.getOriginFieldName(),
	}
}

func (ptf *TypedField) getOriginFieldName() string {
	if ptf.originFieldName == "" {
		return ptf.fieldName
	}
	return ptf.originFieldName
}

var _ Field = (*TypedField)(nil)
