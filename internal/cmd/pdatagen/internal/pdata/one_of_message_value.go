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
	val := &{{ .originFieldPackageName }}.{{ .fieldName }}{}
	ms.orig.{{ .originOneOfFieldName }} = &{{ .originStructType }}{{ "{" }}{{ .fieldName }}: val}
	return new{{ .returnType }}(val, ms.state)
}`

const oneOfMessageAccessorsTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	ms.SetEmpty{{ .fieldName }}()
	assert.Equal(t, New{{ .returnType }}(), ms.{{ .fieldName }}())
	internal.FillOrigTest{{ .returnType }}(ms.orig.Get{{ .originOneOfFieldName }}().(*{{ .originStructType }}).{{ .fieldName }})
	assert.Equal(t, {{ .typeName }}, ms.{{ .originOneOfTypeFuncName }}())
	assert.Equal(t, generateTest{{ .returnType }}(), ms.{{ .fieldName }}())
	sharedState := internal.NewState()
	sharedState.MarkReadOnly()
	assert.Panics(t, func() { new{{ .structName }}(&{{ .originStructName }}{}, sharedState).SetEmpty{{ .fieldName }}() })
}
`

const oneOfMessageSetTestTemplate = `orig.{{ .originOneOfFieldName }} = &{{ .originStructType }}{ 
{{- .fieldName }}: &{{ .originFieldPackageName }}.{{ .fieldName }}{}}
FillOrigTest{{ .fieldOriginName }}(orig.Get{{ .returnType }}())`

const oneOfMessageTestValuesTemplate = `
"oneof_{{ .lowerFieldName }}": { {{ .originOneOfFieldName }}: func() *{{ .originStructType }}{
	val := &{{ .originFieldPackageName }}.{{ .fieldName }}{}
	FillOrigTest{{ .fieldOriginName }}(val)
	return &{{ .originStructType }}{{ "{" }}{{ .fieldName }}: val}
}()},`

const oneOfMessageCopyOrigTemplate = `	case *{{ .originStructType }}:
		{{ .lowerFieldName }} := &{{ .originFieldPackageName}}.{{ .fieldName }}{}
		CopyOrig{{ .fieldOriginName }}({{ .lowerFieldName }}, t.{{ .fieldName }})
		dest.{{ .originOneOfFieldName }} = &{{ .originStructType }}{
			{{ .fieldName }}: {{ .lowerFieldName }},
		}`

const oneOfMessageTypeTemplate = `case *{{ .originStructType }}:
	return {{ .typeName }}`

const oneOfMessageUnmarshalJSONTemplate = `case "{{ lowerFirst .originFieldName }}"{{ if needSnake .originFieldName -}}, "{{ toSnake .originFieldName }}"{{- end }}:
	val := &{{ .originFieldPackageName }}.{{ .fieldName }}{}
	orig.{{ .originOneOfFieldName }} = &{{ .originStructType }}{{ "{" }}{{ .fieldName }}: val}
	UnmarshalJSONOrig{{ .fieldOriginName }}(val, iter)`

type OneOfMessageValue struct {
	fieldName              string
	protoID                uint32
	originFieldPackageName string
	returnMessage          *messageStruct
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

func (omv *OneOfMessageValue) GenerateSetWithTestValue(ms *messageStruct, of *OneOfField) string {
	t := template.Parse("oneOfMessageSetTestTemplate", []byte(oneOfMessageSetTestTemplate))
	return template.Execute(t, omv.templateFields(ms, of))
}

func (omv *OneOfMessageValue) GenerateTestValue(ms *messageStruct, of *OneOfField) string {
	t := template.Parse("oneOfMessageTestValuesTemplate", []byte(oneOfMessageTestValuesTemplate))
	return template.Execute(t, omv.templateFields(ms, of))
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
	return omv.toProtoField(ms, of, true).GenMarshalJSON()
}

func (omv *OneOfMessageValue) GenerateUnmarshalJSON(ms *messageStruct, of *OneOfField) string {
	t := template.Parse("oneOfMessageUnmarshalJSONTemplate", []byte(oneOfMessageUnmarshalJSONTemplate))
	return template.Execute(t, omv.templateFields(ms, of))
}

func (omv *OneOfMessageValue) GenerateSizeProto(ms *messageStruct, of *OneOfField) string {
	return omv.toProtoField(ms, of, true).GenSizeProto()
}

func (omv *OneOfMessageValue) GenerateMarshalProto(ms *messageStruct, of *OneOfField) string {
	return omv.toProtoField(ms, of, true).GenMarshalProto()
}

func (omv *OneOfMessageValue) GenerateUnmarshalProto(ms *messageStruct, of *OneOfField) string {
	return omv.toProtoField(ms, of, false).GenUnmarshalProto()
}

func (omv *OneOfMessageValue) toProtoField(ms *messageStruct, of *OneOfField, oldOneOf bool) *proto.Field {
	pf := &proto.Field{
		Type:                 proto.TypeMessage,
		ID:                   omv.protoID,
		OneOfGroup:           of.originFieldName,
		OneOfMessageFullName: ms.originFullName + "_" + omv.fieldName,
		Name:                 omv.fieldName,
		MessageFullName:      omv.returnMessage.getOriginName(),
		Nullable:             true,
	}
	// TODO: Cleanup this by moving everyone to the new OneOfGroup
	if oldOneOf {
		pf.Name = of.originFieldName + ".(*" + ms.originFullName + "_" + omv.fieldName + ")" + "." + omv.fieldName
	}
	return pf
}

func (omv *OneOfMessageValue) templateFields(ms *messageStruct, of *OneOfField) map[string]any {
	return map[string]any{
		"fieldName":               omv.fieldName,
		"originFieldName":         omv.fieldName,
		"originOneOfFieldName":    of.originFieldName,
		"fieldOriginName":         omv.returnMessage.getOriginName(),
		"typeName":                of.typeName + omv.fieldName,
		"structName":              ms.getName(),
		"returnType":              omv.returnMessage.getName(),
		"originOneOfTypeFuncName": of.typeFuncName(),
		"lowerFieldName":          strings.ToLower(omv.fieldName),
		"originFieldPackageName":  omv.originFieldPackageName,
		"originStructName":        ms.originFullName,
		"originStructType":        ms.originFullName + "_" + omv.fieldName,
	}
}

var _ oneOfValue = (*OneOfMessageValue)(nil)
