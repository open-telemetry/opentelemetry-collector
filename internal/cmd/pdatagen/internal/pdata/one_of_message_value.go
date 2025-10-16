// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	"strings"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const oneOfMessageAccessorsTemplate = `// {{ .fieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
//
// Calling this function when {{ .originOneOfTypeFuncName }}() != {{ .typeName }} returns an invalid
// zero-initialized instance of {{ .returnType }}. Note that using such {{ .returnType }} instance can cause panic.
//
// Calling this function on zero-initialized {{ .structName }} will cause a panic.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .returnType }} {
	v, ok := ms.orig.Get{{ .originOneOfFieldName }}().(*{{ .originStructType }})
	if !ok {
		return {{ .returnType }}{}
	}
	return new{{ .returnType }}(v.{{ .fieldName }}, ms.state)
}

// SetEmpty{{ .fieldName }} sets an empty {{ .lowerFieldName }} to this {{ .structName }}.
//
// After this, {{ .originOneOfTypeFuncName }}() function will return {{ .typeName }}".
//
// Calling this function on zero-initialized {{ .structName }} will cause a panic.
func (ms {{ .structName }}) SetEmpty{{ .fieldName }}() {{ .returnType }} {
	ms.state.AssertMutable()
	var ov *{{ .originStructType }}
	if !internal.UseProtoPooling.IsEnabled() {
		ov = &{{ .originStructType }}{}
	} else {
		ov = internal.ProtoPool{{ .oneOfName }}.Get().(*{{ .originStructType }})
	}
	ov.{{ .fieldName }} = internal.NewOrig{{ .fieldOriginName }}()
	ms.orig.{{ .originOneOfFieldName }} = ov
	return new{{ .returnType }}(ov.{{ .fieldName }}, ms.state)
}`

const oneOfMessageAccessorsTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	ms.SetEmpty{{ .fieldName }}()
	assert.Equal(t, New{{ .returnType }}(), ms.{{ .fieldName }}())
	ms.orig.Get{{ .originOneOfFieldName }}().(*{{ .originStructType }}).{{ .fieldName }} = internal.GenTestOrig{{ .returnType }}()
	assert.Equal(t, {{ .typeName }}, ms.{{ .originOneOfTypeFuncName }}())
	assert.Equal(t, generateTest{{ .returnType }}(), ms.{{ .fieldName }}())
	sharedState := internal.NewState()
	sharedState.MarkReadOnly()
	assert.Panics(t, func() { new{{ .structName }}(&{{ .originStructName }}{}, sharedState).SetEmpty{{ .fieldName }}() })
}
`

const oneOfMessageSetTestTemplate = `orig.{{ .originOneOfFieldName }} = &{{ .originStructType }}{ 
{{- .fieldName }}: GenTestOrig{{ .fieldOriginName }}() }`

const oneOfMessageCopyOrigTemplate = `	case *{{ .originStructType }}:
		var ov *{{ .originStructType }}
		if !UseProtoPooling.IsEnabled() {
			ov = &{{ .originStructType }}{}
		} else {
			ov = ProtoPool{{ .oneOfName }}.Get().(*{{ .originStructType }})
		}
		ov.{{ .fieldName }} = NewOrig{{ .fieldOriginName }}()
		CopyOrig{{ .fieldOriginName }}(ov.{{ .fieldName }}, t.{{ .fieldName }})
		dest.{{ .originOneOfFieldName }} = ov`

const oneOfMessageTypeTemplate = `case *{{ .originStructType }}:
	return {{ .typeName }}`

type OneOfMessageValue struct {
	fieldName     string
	protoID       uint32
	returnMessage *messageStruct
}

func (omv *OneOfMessageValue) GetOriginFieldName() string {
	return omv.fieldName
}

func (omv *OneOfMessageValue) GenerateAccessors(ms *messageStruct, of *OneOfField) string {
	t := template.Parse("oneOfMessageAccessorsTemplate", []byte(oneOfMessageAccessorsTemplate))
	return template.Execute(t, omv.templateFields(ms, of))
}

func (omv *OneOfMessageValue) GenerateTests(ms *messageStruct, of *OneOfField) string {
	t := template.Parse("oneOfMessageAccessorsTestTemplate", []byte(oneOfMessageAccessorsTestTemplate))
	return template.Execute(t, omv.templateFields(ms, of))
}

func (omv *OneOfMessageValue) GenerateTestValue(ms *messageStruct, of *OneOfField) string {
	t := template.Parse("oneOfMessageSetTestTemplate", []byte(oneOfMessageSetTestTemplate))
	return template.Execute(t, omv.templateFields(ms, of))
}

func (omv *OneOfMessageValue) GenerateTestFailingUnmarshalProtoValues(ms *messageStruct, of *OneOfField) string {
	return omv.toProtoField(ms, of).GenTestFailingUnmarshalProtoValues()
}

func (omv *OneOfMessageValue) GenerateTestEncodingValues(ms *messageStruct, of *OneOfField) string {
	return omv.toProtoField(ms, of).GenTestEncodingValues()
}

func (omv *OneOfMessageValue) GeneratePoolOrig(ms *messageStruct, of *OneOfField) string {
	return omv.toProtoField(ms, of).GenPoolVarOrig()
}

func (omv *OneOfMessageValue) GenerateDeleteOrig(ms *messageStruct, of *OneOfField) string {
	return omv.toProtoField(ms, of).GenDeleteOrig()
}

func (omv *OneOfMessageValue) GenerateCopyOrig(ms *messageStruct, of *OneOfField) string {
	t := template.Parse("oneOfMessageCopyOrigTemplate", []byte(oneOfMessageCopyOrigTemplate))
	return template.Execute(t, omv.templateFields(ms, of))
}

func (omv *OneOfMessageValue) GenerateType(ms *messageStruct, of *OneOfField) string {
	t := template.Parse("oneOfMessageTypeTemplate", []byte(oneOfMessageTypeTemplate))
	return template.Execute(t, omv.templateFields(ms, of))
}

func (omv *OneOfMessageValue) GenerateMarshalJSON(ms *messageStruct, of *OneOfField) string {
	return omv.toProtoField(ms, of).GenMarshalJSON()
}

func (omv *OneOfMessageValue) GenerateUnmarshalJSON(ms *messageStruct, of *OneOfField) string {
	return omv.toProtoField(ms, of).GenUnmarshalJSON()
}

func (omv *OneOfMessageValue) GenerateSizeProto(ms *messageStruct, of *OneOfField) string {
	return omv.toProtoField(ms, of).GenSizeProto()
}

func (omv *OneOfMessageValue) GenerateMarshalProto(ms *messageStruct, of *OneOfField) string {
	return omv.toProtoField(ms, of).GenMarshalProto()
}

func (omv *OneOfMessageValue) GenerateUnmarshalProto(ms *messageStruct, of *OneOfField) string {
	return omv.toProtoField(ms, of).GenUnmarshalProto()
}

func (omv *OneOfMessageValue) toProtoField(ms *messageStruct, of *OneOfField) *proto.Field {
	return &proto.Field{
		Type:                 proto.TypeMessage,
		ID:                   omv.protoID,
		OneOfGroup:           of.originFieldName,
		OneOfMessageFullName: ms.originFullName + "_" + omv.fieldName,
		Name:                 omv.fieldName,
		MessageFullName:      omv.returnMessage.getOriginFullName(),
		Nullable:             true,
	}
}

func (omv *OneOfMessageValue) templateFields(ms *messageStruct, of *OneOfField) map[string]any {
	return map[string]any{
		"fieldName":               omv.fieldName,
		"originOneOfFieldName":    of.originFieldName,
		"fieldOriginName":         omv.returnMessage.getOriginName(),
		"typeName":                of.typeName + omv.fieldName,
		"structName":              ms.getName(),
		"returnType":              omv.returnMessage.getName(),
		"originOneOfTypeFuncName": of.typeFuncName(),
		"lowerFieldName":          strings.ToLower(omv.fieldName),
		"originStructName":        ms.originFullName,
		"originStructType":        ms.originFullName + "_" + omv.fieldName,
		"oneOfName":               proto.ExtractNameFromFull(ms.originFullName + "_" + omv.fieldName),
	}
}

var _ oneOfValue = (*OneOfMessageValue)(nil)
