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
	ms.orig.{{ .originFieldName }} = {{ .messageType }}(v)
}`

const typedAccessorsTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	assert.Equal(t, {{ .packageName }}{{ .returnType }}({{ .defaultVal }}), ms.{{ .fieldName }}())
	testVal{{ .fieldName }} := {{ .packageName }}{{ .returnType }}({{ .testValue }})
	ms.Set{{ .fieldName }}(testVal{{ .fieldName }})
	assert.Equal(t, testVal{{ .fieldName }}, ms.{{ .fieldName }}())
}`

const typedSetTestTemplate = `orig.{{ .originFieldName }} = {{ .testValue }}`

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

type ProtoTypedField struct {
	*proto.Field
}

func (ptf ProtoTypedField) GenMarshalJSON() string {
	if ptf.MessageName == "TraceID" || ptf.MessageName == "SpanID" || ptf.MessageName == "ProfileID" {
		return "if !orig." + ptf.Name + ".IsEmpty() {\n" + ptf.Field.GenMarshalJSON() + "\n}"
	}
	return ptf.Field.GenMarshalJSON()
}

func (ptf *TypedField) toProtoField(ms *messageStruct) proto.FieldInterface {
	return ProtoTypedField{&proto.Field{
		Type:              ptf.returnType.protoType,
		ID:                ptf.protoID,
		Name:              ptf.getOriginFieldName(),
		MessageName:       ptf.returnType.messageName,
		ParentMessageName: ms.protoName,
	}}
}

func (ptf *TypedField) getOriginFieldName() string {
	if ptf.originFieldName == "" {
		return ptf.fieldName
	}
	return ptf.originFieldName
}

func (ptf *TypedField) templateFields(ms *messageStruct) map[string]any {
	pf := ptf.toProtoField(ms)
	messageType := pf.GoType()
	defaultVal := ptf.returnType.defaultVal
	testVal := ptf.returnType.testVal
	if ptf.returnType.protoType == proto.TypeMessage || ptf.returnType.protoType == proto.TypeEnum {
		messageType = "internal." + messageType
		defaultVal = "internal." + defaultVal
		testVal = "internal." + testVal
	}
	return map[string]any{
		"structName": ms.getName(),
		"defaultVal": defaultVal,
		"packageName": func() string {
			if ptf.returnType.packageName != ms.packageName {
				return ptf.returnType.packageName + "."
			}
			return ""
		}(),
		"hasWrapper":      usedByOtherDataTypes(ptf.returnType.packageName),
		"returnType":      ptf.returnType.structName,
		"fieldName":       ptf.fieldName,
		"originFieldName": ptf.getOriginFieldName(),
		"lowerFieldName":  strings.ToLower(ptf.fieldName),
		"testValue":       testVal,
		"messageType":     messageType,
	}
}

var _ Field = (*TypedField)(nil)
