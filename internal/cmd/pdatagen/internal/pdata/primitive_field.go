// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	"strings"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
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
	sharedState := internal.NewState()
	sharedState.MarkReadOnly()
	assert.Panics(t, func() { new{{ .structName }}(&{{ .originStructName }}{}, sharedState).Set{{ .fieldName }}({{ .testValue }}) })
}`

const primitiveSetTestTemplate = `orig.{{ .originFieldName }} = {{ .testValue }}`

const primitiveCopyOrigTemplate = `dest.{{ .originFieldName }} = src.{{ .originFieldName }}`

type PrimitiveField struct {
	fieldName string
	protoType proto.Type
	protoID   uint32
}

func (pf *PrimitiveField) GenerateAccessors(ms *messageStruct) string {
	t := template.Parse("primitiveAccessorsTemplate", []byte(primitiveAccessorsTemplate))
	return template.Execute(t, pf.templateFields(ms))
}

func (pf *PrimitiveField) GenerateAccessorsTest(ms *messageStruct) string {
	t := template.Parse("primitiveAccessorsTestTemplate", []byte(primitiveAccessorsTestTemplate))
	return template.Execute(t, pf.templateFields(ms))
}

func (pf *PrimitiveField) GenerateTestValue(ms *messageStruct) string {
	t := template.Parse("primitiveSetTestTemplate", []byte(primitiveSetTestTemplate))
	return template.Execute(t, pf.templateFields(ms))
}

func (pf *PrimitiveField) GenerateTestFailingUnmarshalProtoValues(*messageStruct) string {
	return pf.toProtoField().GenTestFailingUnmarshalProtoValues()
}

func (pf *PrimitiveField) GenerateTestEncodingValues(*messageStruct) string {
	return pf.toProtoField().GenTestEncodingValues()
}

func (pf *PrimitiveField) GeneratePoolOrig(*messageStruct) string {
	return ""
}

func (pf *PrimitiveField) GenerateDeleteOrig(*messageStruct) string {
	return pf.toProtoField().GenDeleteOrig()
}

func (pf *PrimitiveField) GenerateCopyOrig(ms *messageStruct) string {
	t := template.Parse("primitiveCopyOrigTemplate", []byte(primitiveCopyOrigTemplate))
	return template.Execute(t, pf.templateFields(ms))
}

func (pf *PrimitiveField) GenerateMarshalJSON(*messageStruct) string {
	return pf.toProtoField().GenMarshalJSON()
}

func (pf *PrimitiveField) GenerateUnmarshalJSON(*messageStruct) string {
	return pf.toProtoField().GenUnmarshalJSON()
}

func (pf *PrimitiveField) GenerateSizeProto(*messageStruct) string {
	return pf.toProtoField().GenSizeProto()
}

func (pf *PrimitiveField) GenerateMarshalProto(*messageStruct) string {
	return pf.toProtoField().GenMarshalProto()
}

func (pf *PrimitiveField) GenerateUnmarshalProto(*messageStruct) string {
	return pf.toProtoField().GenUnmarshalProto()
}

func (pf *PrimitiveField) toProtoField() *proto.Field {
	return &proto.Field{
		Type: pf.protoType,
		ID:   pf.protoID,
		Name: pf.fieldName,
	}
}

func (pf *PrimitiveField) templateFields(ms *messageStruct) map[string]any {
	prf := pf.toProtoField()
	return map[string]any{
		"structName":       ms.getName(),
		"packageName":      "",
		"defaultVal":       prf.DefaultValue(),
		"fieldName":        pf.fieldName,
		"lowerFieldName":   strings.ToLower(pf.fieldName),
		"testValue":        prf.TestValue(),
		"returnType":       prf.GoType(),
		"origAccessor":     origAccessor(ms.getHasWrapper()),
		"stateAccessor":    stateAccessor(ms.getHasWrapper()),
		"originStructName": ms.originFullName,
		"originFieldName":  pf.fieldName,
	}
}

var _ Field = (*PrimitiveField)(nil)
