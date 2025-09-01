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
// {{ .fieldName }} value otherwise.
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

type OptionalPrimitiveField struct {
	fieldName string
	protoID   uint32
	protoType proto.Type
}

func (opv *OptionalPrimitiveField) GenerateAccessors(ms *messageStruct) string {
	return template.Execute(template.Parse("optionalPrimitiveAccessorsTemplate", []byte(optionalPrimitiveAccessorsTemplate)), opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) GenerateAccessorsTest(ms *messageStruct) string {
	return template.Execute(template.Parse("optionalPrimitiveAccessorsTestTemplate", []byte(optionalPrimitiveAccessorsTestTemplate)), opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) GenerateTestValue(ms *messageStruct) string {
	return template.Execute(template.Parse("optionalPrimitiveSetTestTemplate", []byte(optionalPrimitiveSetTestTemplate)), opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) GenerateTestFailingUnmarshalProtoValues(ms *messageStruct) string {
	return opv.toProtoField(ms).GenTestFailingUnmarshalProtoValues()
}

func (opv *OptionalPrimitiveField) GenerateTestEncodingValues(ms *messageStruct) string {
	return opv.toProtoField(ms).GenTestEncodingValues()
}

func (opv *OptionalPrimitiveField) GeneratePoolOrig(ms *messageStruct) string {
	return opv.toProtoField(ms).GenPoolVarOrig()
}

func (opv *OptionalPrimitiveField) GenerateDeleteOrig(ms *messageStruct) string {
	return "switch ov := orig." + opv.fieldName + "_.(type) {\n\tcase *" + ms.getOriginFullName() + "_" + opv.fieldName + ":\n\t" + opv.toProtoField(ms).GenDeleteOrig() + "\n}\n"
}

func (opv *OptionalPrimitiveField) GenerateCopyOrig(ms *messageStruct) string {
	return template.Execute(template.Parse("optionalPrimitiveCopyOrigTemplate", []byte(optionalPrimitiveCopyOrigTemplate)), opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) GenerateMarshalJSON(ms *messageStruct) string {
	return "if orig, ok := orig." + opv.fieldName + "_.(*" + ms.originFullName + "_" + opv.fieldName + "); ok {\n\t" + opv.toProtoField(ms).GenMarshalJSON() + "}"
}

func (opv *OptionalPrimitiveField) GenerateUnmarshalJSON(ms *messageStruct) string {
	return opv.toProtoField(ms).GenUnmarshalJSON()
}

func (opv *OptionalPrimitiveField) GenerateSizeProto(ms *messageStruct) string {
	return "if orig, ok := orig." + opv.fieldName + "_.(*" + ms.originFullName + "_" + opv.fieldName + "); ok {\n\t_ = orig\n\t" + opv.toProtoField(ms).GenSizeProto() + "}"
}

func (opv *OptionalPrimitiveField) GenerateMarshalProto(ms *messageStruct) string {
	return "if orig, ok := orig." + opv.fieldName + "_.(*" + ms.originFullName + "_" + opv.fieldName + "); ok {\n\t" + opv.toProtoField(ms).GenMarshalProto() + "}"
}

func (opv *OptionalPrimitiveField) GenerateUnmarshalProto(ms *messageStruct) string {
	return opv.toProtoField(ms).GenUnmarshalProto()
}

func (opv *OptionalPrimitiveField) toProtoField(ms *messageStruct) *proto.Field {
	return &proto.Field{
		Type:                 opv.protoType,
		ID:                   opv.protoID,
		OneOfGroup:           opv.fieldName + "_",
		Name:                 opv.fieldName,
		OneOfMessageFullName: ms.originFullName + "_" + opv.fieldName,
		Nullable:             true,
	}
}

func (opv *OptionalPrimitiveField) templateFields(ms *messageStruct) map[string]any {
	pf := opv.toProtoField(ms)
	return map[string]any{
		"structName":       ms.getName(),
		"defaultVal":       pf.DefaultValue(),
		"fieldName":        opv.fieldName,
		"lowerFieldName":   strings.ToLower(opv.fieldName),
		"testValue":        pf.TestValue(),
		"returnType":       pf.GoType(),
		"originName":       ms.getOriginName(),
		"originStructName": ms.getOriginFullName(),
		"originStructType": ms.getOriginFullName() + "_" + opv.fieldName,
	}
}

var _ Field = (*OptionalPrimitiveField)(nil)
