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
	v, ok := ms.orig.Get{{ .originOneOfFieldName }}().(*internal.{{ .originStructType }})
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
	var ov *internal.{{ .originStructType }}
	if !internal.UseProtoPooling.IsEnabled() {
		ov = &internal.{{ .originStructType }}{}
	} else {
		ov = internal.ProtoPool{{ .oneOfName }}.Get().(*internal.{{ .originStructType }})
	}
	ov.{{ .fieldName }} = internal.New{{ .fieldOriginName }}()
	ms.orig.{{ .originOneOfFieldName }} = ov
	return new{{ .returnType }}(ov.{{ .fieldName }}, ms.state)
}`

const oneOfMessageAccessorsTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	ms.SetEmpty{{ .fieldName }}()
	assert.Equal(t, New{{ .returnType }}(), ms.{{ .fieldName }}())
	ms.orig.Get{{ .originOneOfFieldName }}().(*internal.{{ .originStructType }}).{{ .fieldName }} = internal.GenTest{{ .returnType }}()
	assert.Equal(t, {{ .typeName }}, ms.{{ .originOneOfTypeFuncName }}())
	assert.Equal(t, generateTest{{ .returnType }}(), ms.{{ .fieldName }}())
	sharedState := internal.NewState()
	sharedState.MarkReadOnly()
	assert.Panics(t, func() { new{{ .structName }}(internal.New{{ .originStructName }}(), sharedState).SetEmpty{{ .fieldName }}() })
}
`

const oneOfMessageSetTestTemplate = `orig.{{ .originOneOfFieldName }} = &internal.{{ .originStructType }}{ 
{{- .fieldName }}: GenTest{{ .fieldOriginName }}() }`

const oneOfMessageTypeTemplate = `case *internal.{{ .originStructType }}:
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

func (omv *OneOfMessageValue) GenerateType(ms *messageStruct, of *OneOfField) string {
	t := template.Parse("oneOfMessageTypeTemplate", []byte(oneOfMessageTypeTemplate))
	return template.Execute(t, omv.templateFields(ms, of))
}

func (omv *OneOfMessageValue) toProtoField(ms *messageStruct, of *OneOfField) proto.FieldInterface {
	return &proto.Field{
		Type:              proto.TypeMessage,
		ID:                omv.protoID,
		OneOfGroup:        of.originFieldName,
		OneOfMessageName:  ms.protoName + "_" + omv.fieldName,
		Name:              omv.fieldName,
		MessageName:       omv.returnMessage.getOriginFullName(),
		ParentMessageName: ms.protoName,
		Nullable:          true,
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
		"originStructName":        ms.protoName,
		"originStructType":        ms.protoName + "_" + omv.fieldName,
		"oneOfName":               proto.ExtractNameFromFull(ms.protoName + "_" + omv.fieldName),
	}
}

var _ oneOfValue = (*OneOfMessageValue)(nil)
