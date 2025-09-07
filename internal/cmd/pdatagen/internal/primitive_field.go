// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"strings"
	"text/template"
)

const primitiveAccessorsTemplate = `// {{ .fieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .packageName }}{{ .returnType }} {
	return ms.{{ .origAccessor }}.{{ .originFieldName }}
}

// Set{{ .fieldName }} replaces the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) Set{{ .fieldName }}(v {{ .returnType }}) {
	ms.{{ .stateAccessor }}.AssertMutable()
	ms.{{ .origAccessor }}.{{ .originFieldName }} = v
}`

const primitiveAccessorsTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	{{- if eq .returnType "bool" }}
	assert.{{- if eq .defaultVal "true" }}True{{- else }}False{{- end }}(t, ms.{{ .fieldName }}())
	{{- else if eq .returnType "float64" }}
	assert.InDelta(t, {{ .defaultVal }}, ms.{{ .fieldName }}(), 0.01)
	{{- else if and (eq .returnType "string") (eq .defaultVal "\"\"") }}
	assert.Empty(t, ms.{{ .fieldName }}())
	{{- else }}
	assert.Equal(t, {{ .defaultVal }}, ms.{{ .fieldName }}())
	{{- end }}
	ms.Set{{ .fieldName }}({{ .testValue }})
	{{- if eq .returnType "bool" }}
	assert.{{- if eq .testValue "true" }}True{{- else }}False{{- end }}(t, ms.{{ .fieldName }}())
	{{- else if eq .returnType "float64"}}
	assert.InDelta(t, {{ .testValue }}, ms.{{ .fieldName }}(), 0.01)
	{{- else if and (eq .returnType "string") (eq .testValue "\"\"") }}
	assert.Empty(t, ms.{{ .fieldName }}())
	{{- else }}
	assert.Equal(t, {{ .testValue }}, ms.{{ .fieldName }}())
	{{- end }}
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { new{{ .structName }}(&{{ .originStructName }}{}, &sharedState).Set{{ .fieldName }}({{ .testValue }}) })
}`

const primitiveSetTestTemplate = `orig.{{ .originFieldName }} = {{ .testValue }}`

const primitiveCopyOrigTemplate = `dest.{{ .originFieldName }} = src.{{ .originFieldName }}`

const primitiveMarshalJSONTemplate = `if orig.{{ .originFieldName }} != {{ .defaultVal }} {
		dest.WriteObjectField("{{ lowerFirst .originFieldName }}")
		dest.Write{{ upperFirst .returnType }}(orig.{{ .originFieldName }})
	}`

const primitiveUnmarshalJSONTemplate = `case "{{ lowerFirst .originFieldName }}"{{ if needSnake .originFieldName -}}, "{{ toSnake .originFieldName }}"{{- end }}:
		orig.{{ .originFieldName }} = iter.Read{{ upperFirst .returnType }}()`

type PrimitiveField struct {
	fieldName string
	protoType ProtoType
	protoID   uint32
}

func (pf *PrimitiveField) GenerateAccessors(ms *messageStruct) string {
	t := template.Must(templateNew("primitiveAccessorsTemplate").Parse(primitiveAccessorsTemplate))
	return executeTemplate(t, pf.templateFields(ms))
}

func (pf *PrimitiveField) GenerateAccessorsTest(ms *messageStruct) string {
	t := template.Must(templateNew("primitiveAccessorsTestTemplate").Parse(primitiveAccessorsTestTemplate))
	return executeTemplate(t, pf.templateFields(ms))
}

func (pf *PrimitiveField) GenerateSetWithTestValue(ms *messageStruct) string {
	t := template.Must(templateNew("primitiveSetTestTemplate").Parse(primitiveSetTestTemplate))
	return executeTemplate(t, pf.templateFields(ms))
}

func (pf *PrimitiveField) GenerateCopyOrig(ms *messageStruct) string {
	t := template.Must(templateNew("primitiveCopyOrigTemplate").Parse(primitiveCopyOrigTemplate))
	return executeTemplate(t, pf.templateFields(ms))
}

func (pf *PrimitiveField) GenerateMarshalJSON(ms *messageStruct) string {
	t := template.Must(templateNew("primitiveMarshalJSONTemplate").Parse(primitiveMarshalJSONTemplate))
	return executeTemplate(t, pf.templateFields(ms))
}

func (pf *PrimitiveField) GenerateUnmarshalJSON(ms *messageStruct) string {
	t := template.Must(templateNew("primitiveUnmarshalJSONTemplate").Parse(primitiveUnmarshalJSONTemplate))
	return executeTemplate(t, pf.templateFields(ms))
}

func (pf *PrimitiveField) GenerateSizeProto(*messageStruct) string {
	return pf.toProtoField().genSizeProto()
}

func (pf *PrimitiveField) GenerateMarshalProto(*messageStruct) string {
	return pf.toProtoField().genMarshalProto()
}

func (pf *PrimitiveField) toProtoField() *ProtoField {
	return &ProtoField{
		Type: pf.protoType,
		ID:   pf.protoID,
		Name: pf.fieldName,
	}
}

func (pf *PrimitiveField) templateFields(ms *messageStruct) map[string]any {
	return map[string]any{
		"structName":       ms.getName(),
		"packageName":      "",
		"defaultVal":       pf.protoType.defaultValue(""),
		"fieldName":        pf.fieldName,
		"lowerFieldName":   strings.ToLower(pf.fieldName),
		"testValue":        pf.protoType.testValue(pf.fieldName),
		"returnType":       pf.protoType.goType(""),
		"origAccessor":     origAccessor(ms.packageName),
		"stateAccessor":    stateAccessor(ms.packageName),
		"originStructName": ms.originFullName,
		"originFieldName":  pf.fieldName,
	}
}

var _ Field = (*PrimitiveField)(nil)
