// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"strings"
	"text/template"
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
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { new{{ .structName }}(&{{ .originStructName }}{}, &sharedState).SetEmpty{{ .fieldName }}() })
}
`

const oneOfMessageSetTestTemplate = `orig.{{ .originOneOfFieldName }} = &{{ .originStructName }}_{{ .fieldName -}}{ 
{{- .fieldName }}: &{{ .originFieldPackageName }}.{{ .fieldName }}{}}
FillOrigTest{{ .fieldOriginName }}(orig.Get{{ .returnType }}())`

const oneOfMessageCopyOrigTemplate = `	case *{{ .originStructType }}:
		{{ .lowerFieldName }} := &{{ .originFieldPackageName}}.{{ .fieldName }}{}
		CopyOrig{{ .fieldOriginName }}({{ .lowerFieldName }}, t.{{ .fieldName }})
		dest.{{ .originOneOfFieldName }} = &{{ .originStructType }}{
			{{ .fieldName }}: {{ .lowerFieldName }},
		}`

const oneOfMessageTypeTemplate = `case *{{ .originStructName }}_{{ .originFieldName }}:
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
	t := template.Must(templateNew("oneOfMessageAccessorsTemplate").Parse(oneOfMessageAccessorsTemplate))
	return executeTemplate(t, omv.templateFields(ms, of))
}

func (omv *OneOfMessageValue) GenerateTests(ms *messageStruct, of *OneOfField) string {
	t := template.Must(templateNew("oneOfMessageAccessorsTestTemplate").Parse(oneOfMessageAccessorsTestTemplate))
	return executeTemplate(t, omv.templateFields(ms, of))
}

func (omv *OneOfMessageValue) GenerateSetWithTestValue(ms *messageStruct, of *OneOfField) string {
	t := template.Must(templateNew("oneOfMessageSetTestTemplate").Parse(oneOfMessageSetTestTemplate))
	return executeTemplate(t, omv.templateFields(ms, of))
}

func (omv *OneOfMessageValue) GenerateCopyOrig(ms *messageStruct, of *OneOfField) string {
	t := template.Must(templateNew("oneOfMessageCopyOrigTemplate").Parse(oneOfMessageCopyOrigTemplate))
	return executeTemplate(t, omv.templateFields(ms, of))
}

func (omv *OneOfMessageValue) GenerateType(ms *messageStruct, of *OneOfField) string {
	t := template.Must(templateNew("oneOfMessageTypeTemplate").Parse(oneOfMessageTypeTemplate))
	return executeTemplate(t, omv.templateFields(ms, of))
}

func (omv *OneOfMessageValue) GenerateMarshalJSON(ms *messageStruct, of *OneOfField) string {
	return omv.toProtoField(ms, of).genMarshalJSON()
}

func (omv *OneOfMessageValue) GenerateUnmarshalJSON(ms *messageStruct, of *OneOfField) string {
	t := template.Must(templateNew("oneOfMessageUnmarshalJSONTemplate").Parse(oneOfMessageUnmarshalJSONTemplate))
	return executeTemplate(t, omv.templateFields(ms, of))
}

func (omv *OneOfMessageValue) GenerateSizeProto(ms *messageStruct, of *OneOfField) string {
	return omv.toProtoField(ms, of).genSizeProto()
}

func (omv *OneOfMessageValue) GenerateMarshalProto(ms *messageStruct, of *OneOfField) string {
	return omv.toProtoField(ms, of).genMarshalProto()
}

func (omv *OneOfMessageValue) toProtoField(ms *messageStruct, of *OneOfField) *ProtoField {
	return &ProtoField{
		Type:        ProtoTypeMessage,
		ID:          omv.protoID,
		Name:        of.originFieldName + ".(*" + ms.originFullName + "_" + omv.fieldName + ")" + "." + omv.fieldName,
		MessageName: omv.returnMessage.getOriginName(),
		Nullable:    true,
	}
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
