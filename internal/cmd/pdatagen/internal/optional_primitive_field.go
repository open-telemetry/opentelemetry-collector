// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"
import (
	"strings"
	"text/template"
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

const optionalPrimitiveTestValuesTemplate = `
"default_{{ .lowerFieldName }}": { {{ .fieldName }}_: &{{ .originStructType }}{{ "{" }}{{ .fieldName }}: {{ .defaultVal }}} },`

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
	protoType ProtoType
}

func (opv *OptionalPrimitiveField) GenerateAccessors(ms *messageStruct) string {
	t := template.Must(templateNew("optionalPrimitiveAccessorsTemplate").Parse(optionalPrimitiveAccessorsTemplate))
	return executeTemplate(t, opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) GenerateAccessorsTest(ms *messageStruct) string {
	t := template.Must(templateNew("optionalPrimitiveAccessorsTestTemplate").Parse(optionalPrimitiveAccessorsTestTemplate))
	return executeTemplate(t, opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) GenerateSetWithTestValue(ms *messageStruct) string {
	t := template.Must(templateNew("optionalPrimitiveSetTestTemplate").Parse(optionalPrimitiveSetTestTemplate))
	return executeTemplate(t, opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) GenerateTestValue(ms *messageStruct) string {
	t := template.Must(templateNew("optionalPrimitiveTestValuesTemplate").Parse(optionalPrimitiveTestValuesTemplate))
	return executeTemplate(t, opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) GenerateCopyOrig(ms *messageStruct) string {
	t := template.Must(templateNew("optionalPrimitiveCopyOrigTemplate").Parse(optionalPrimitiveCopyOrigTemplate))
	return executeTemplate(t, opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) GenerateMarshalJSON(ms *messageStruct) string {
	return "if orig." + opv.fieldName + "_ != nil {\n\t" + opv.toProtoField(ms).genMarshalJSON() + "\n}"
}

func (opv *OptionalPrimitiveField) GenerateUnmarshalJSON(ms *messageStruct) string {
	t := template.Must(templateNew("optionalPrimitiveUnmarshalJSONTemplate").Parse(optionalPrimitiveUnmarshalJSONTemplate))
	return executeTemplate(t, opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) GenerateSizeProto(ms *messageStruct) string {
	return "if orig." + opv.fieldName + "_ != nil {\n\t" + opv.toProtoField(ms).genSizeProto() + "\n}"
}

func (opv *OptionalPrimitiveField) GenerateMarshalProto(ms *messageStruct) string {
	return "if orig." + opv.fieldName + "_ != nil {\n\t" + opv.toProtoField(ms).genMarshalProto() + "\n}"
}

func (opv *OptionalPrimitiveField) toProtoField(ms *messageStruct) *ProtoField {
	return &ProtoField{
		Type:     opv.protoType,
		ID:       opv.protoID,
		Name:     opv.fieldName + "_.(*" + ms.originFullName + "_" + opv.fieldName + ")" + "." + opv.fieldName,
		Nullable: true,
	}
}

func (opv *OptionalPrimitiveField) templateFields(ms *messageStruct) map[string]any {
	return map[string]any{
		"structName":       ms.getName(),
		"packageName":      "",
		"defaultVal":       opv.protoType.defaultValue(""),
		"fieldName":        opv.fieldName,
		"lowerFieldName":   strings.ToLower(opv.fieldName),
		"testValue":        opv.protoType.testValue(opv.fieldName),
		"returnType":       opv.protoType.goType(""),
		"originStructName": ms.originFullName,
		"originStructType": ms.originFullName + "_" + opv.fieldName,
	}
}

var _ Field = (*OptionalPrimitiveField)(nil)
