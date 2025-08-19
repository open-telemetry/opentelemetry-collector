// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	"strings"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const optionalPrimitiveAccessorsTemplate = `// {{ .fieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .returnType }} {
	return ms.orig.Get{{ .fieldName }}()
}

// Has{{ .fieldName }} returns true if the {{ .structName }} contains a
// {{ .fieldName }} value, false otherwise.
func (ms {{ .structName }}) Has{{ .fieldName }}() bool {
	return ms.orig.{{ .fieldName }}_ != nil
}

// Set{{ .fieldName }} replaces the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) Set{{ .fieldName }}(v {{ .returnType }}) {
	ms.state.AssertMutable()
	ms.orig.{{ .fieldName }}_ = &{{ .originStructType }}{{ "{" }}{{ .fieldName }}: v}
}

// Remove{{ .fieldName }} removes the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) Remove{{ .fieldName }}() {
	ms.state.AssertMutable()
	ms.orig.{{ .fieldName }}_ = nil
}`

const optionalPrimitiveAccessorsTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	{{- if eq .returnType "float64" }}
	assert.InDelta(t, {{ .defaultVal }}, ms.{{ .fieldName }}() , 0.01)
	{{- else }}
	assert.Equal(t, {{ .defaultVal }}, ms.{{ .fieldName }}())
	{{- end }}
	ms.Set{{ .fieldName }}({{ .testValue }})
	assert.True(t, ms.Has{{ .fieldName }}())
	{{- if eq .returnType "float64" }}
	assert.InDelta(t, {{.testValue }}, ms.{{ .fieldName }}(), 0.01)
	{{- else }}
	assert.Equal(t, {{ .testValue }}, ms.{{ .fieldName }}())
	{{- end }}
	ms.Remove{{ .fieldName }}()
	assert.False(t, ms.Has{{ .fieldName }}())
	dest := New{{ .structName }}()
	dest.Set{{ .fieldName }}({{ .testValue }})
	ms.CopyTo(dest)
	assert.False(t, dest.Has{{ .fieldName }}())
}`

const optionalPrimitiveSetTestTemplate = `orig.{{ .fieldName }}_ = &{{ .originStructType }}{
{{- .fieldName }}: {{ .testValue }}}`

const optionalPrimitiveCopyOrigTemplate = `if src{{ .fieldName }}, ok := src.{{ .fieldName }}_.(*{{ .originStructType }}); ok {
	dest{{ .fieldName }}, ok := dest.{{ .fieldName }}_.(*{{ .originStructType }})
	if !ok {
		dest{{ .fieldName }} = &{{ .originStructType }}{}
		dest.{{ .fieldName }}_ = dest{{ .fieldName }}
	}
	dest{{ .fieldName }}.{{ .fieldName }} = src{{ .fieldName }}.{{ .fieldName }}
} else {
	dest.{{ .fieldName }}_ = nil
}`

const optionalPrimitiveUnmarshalJSONTemplate = `case "{{ lowerFirst .fieldName }}"{{ if needSnake .fieldName -}}, "{{ toSnake .fieldName }}"{{- end }}:
		orig.{{ .fieldName }}_ = &{{ .originStructType }}{{ "{" }}{{ .fieldName }}: iter.Read{{ upperFirst .returnType }}()}`

type OptionalPrimitiveField struct {
	fieldName string
	protoID   uint32
	protoType proto.Type
}

func (opv *OptionalPrimitiveField) GenerateAccessors(ms *messageStruct) string {
	t := template.Parse("optionalPrimitiveAccessorsTemplate", []byte(optionalPrimitiveAccessorsTemplate))
	return template.Execute(t, opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) GenerateAccessorsTest(ms *messageStruct) string {
	t := template.Parse("optionalPrimitiveAccessorsTestTemplate", []byte(optionalPrimitiveAccessorsTestTemplate))
	return template.Execute(t, opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) GenerateTestValue(ms *messageStruct) string {
	t := template.Parse("optionalPrimitiveSetTestTemplate", []byte(optionalPrimitiveSetTestTemplate))
	return template.Execute(t, opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) GenerateTestEncodingValues(ms *messageStruct) string {
	return opv.toProtoField(ms, false).GenTestEncodingValues()
}

func (opv *OptionalPrimitiveField) GenerateCopyOrig(ms *messageStruct) string {
	t := template.Parse("optionalPrimitiveCopyOrigTemplate", []byte(optionalPrimitiveCopyOrigTemplate))
	return template.Execute(t, opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) GenerateMarshalJSON(ms *messageStruct) string {
	return "if orig." + opv.fieldName + "_ != nil {\n\t" + opv.toProtoField(ms, true).GenMarshalJSON() + "\n}"
}

func (opv *OptionalPrimitiveField) GenerateUnmarshalJSON(ms *messageStruct) string {
	t := template.Parse("optionalPrimitiveUnmarshalJSONTemplate", []byte(optionalPrimitiveUnmarshalJSONTemplate))
	return template.Execute(t, opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) GenerateSizeProto(ms *messageStruct) string {
	return "if orig." + opv.fieldName + "_ != nil {\n\t" + opv.toProtoField(ms, true).GenSizeProto() + "\n}"
}

func (opv *OptionalPrimitiveField) GenerateMarshalProto(ms *messageStruct) string {
	return "if orig." + opv.fieldName + "_ != nil {\n\t" + opv.toProtoField(ms, true).GenMarshalProto() + "\n}"
}

func (opv *OptionalPrimitiveField) GenerateUnmarshalProto(ms *messageStruct) string {
	return opv.toProtoField(ms, false).GenUnmarshalProto()
}

func (opv *OptionalPrimitiveField) toProtoField(ms *messageStruct, oldOneOf bool) *proto.Field {
	pf := &proto.Field{
		Type:                 opv.protoType,
		ID:                   opv.protoID,
		OneOfGroup:           opv.fieldName + "_",
		Name:                 opv.fieldName,
		OneOfMessageFullName: ms.originFullName + "_" + opv.fieldName,
		Nullable:             true,
	}
	// TODO: Cleanup this by moving everyone to the new OneOfGroup
	if oldOneOf {
		pf.Name = opv.fieldName + "_.(*" + ms.originFullName + "_" + opv.fieldName + ")" + "." + opv.fieldName
	}
	return pf
}

func (opv *OptionalPrimitiveField) templateFields(ms *messageStruct) map[string]any {
	pf := opv.toProtoField(ms, false)
	return map[string]any{
		"structName":       ms.getName(),
		"packageName":      "",
		"defaultVal":       pf.DefaultValue(),
		"fieldName":        opv.fieldName,
		"lowerFieldName":   strings.ToLower(opv.fieldName),
		"testValue":        pf.TestValue(),
		"returnType":       pf.GoType(),
		"originStructName": ms.originFullName,
		"originStructType": ms.originFullName + "_" + opv.fieldName,
	}
}

var _ Field = (*OptionalPrimitiveField)(nil)
