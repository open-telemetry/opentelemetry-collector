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
	ms.orig.{{ .fieldName }}_ = &internal.{{ .originStructType }}{{ "{" }}{{ .fieldName }}: v}
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

const optionalPrimitiveSetTestTemplate = `orig.{{ .fieldName }}_ = &internal.{{ .originStructType }}{
{{- .fieldName }}: {{ .testValue }}}`

const optionalOneOfMessageOrigTemplate = `
func (m *{{ .ParentMessageName }}) Get{{ .OneOfGroup }}() any {
	if m != nil {
		return m.{{ .Name }}_
	}
	return nil
}

`

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

type optionalPrimitiveProtoField struct {
	*proto.Field
}

func (opv optionalPrimitiveProtoField) GetName() string {
	return opv.Name + "_"
}

func (opv optionalPrimitiveProtoField) TestValue() string {
	return "&" + opv.OneOfMessageName + "{" + opv.Name + ": " + opv.Field.TestValue() + "}"
}

func (opv optionalPrimitiveProtoField) GenMessageField() string {
	return opv.Name + "_ any"
}

func (opv optionalPrimitiveProtoField) GenOneOfMessages() string {
	return template.Execute(template.Parse("optionalOneOfMessageOrigTemplate", []byte(optionalOneOfMessageOrigTemplate)), opv.Field) + opv.Field.GenOneOfMessages()
}

func (opv optionalPrimitiveProtoField) GenCopy() string {
	return "switch t := src." + opv.Name + "_.(type) {\n\tcase *" + opv.OneOfMessageName + ":\n\t" + opv.Field.GenCopy() + "\ndefault: dest." + opv.Name + "_ = nil\n}\n"
}

func (opv optionalPrimitiveProtoField) GenDelete() string {
	return "switch ov := orig." + opv.Name + "_.(type) {\n\tcase *" + opv.OneOfMessageName + ":\n\t" + opv.Field.GenDelete() + "\n}\n"
}

func (opv optionalPrimitiveProtoField) GenMarshalJSON() string {
	return "if orig, ok := orig." + opv.Name + "_.(*" + opv.OneOfMessageName + "); ok {\n\t" + opv.Field.GenMarshalJSON() + "}"
}

func (opv optionalPrimitiveProtoField) GenSizeProto() string {
	return "if orig, ok := orig." + opv.Name + "_.(*" + opv.OneOfMessageName + "); ok {\n\t_ = orig\n\t" + opv.Field.GenSizeProto() + "}"
}

func (opv optionalPrimitiveProtoField) GenMarshalProto() string {
	return "if orig, ok := orig." + opv.Name + "_.(*" + opv.OneOfMessageName + "); ok {\n\t" + opv.Field.GenMarshalProto() + "}"
}

func (opv *OptionalPrimitiveField) toProtoField(ms *messageStruct) proto.FieldInterface {
	return optionalPrimitiveProtoField{&proto.Field{
		Type:              opv.protoType,
		ID:                opv.protoID,
		OneOfGroup:        opv.fieldName + "_",
		Name:              opv.fieldName,
		OneOfMessageName:  ms.protoName + "_" + opv.fieldName,
		ParentMessageName: ms.protoName,
		Nullable:          true,
	}}
}

func (opv *OptionalPrimitiveField) templateFields(ms *messageStruct) map[string]any {
	pf := opv.toProtoField(ms).(optionalPrimitiveProtoField)
	return map[string]any{
		"structName":       ms.getName(),
		"defaultVal":       pf.DefaultValue(),
		"fieldName":        opv.fieldName,
		"lowerFieldName":   strings.ToLower(opv.fieldName),
		"testValue":        pf.Field.TestValue(),
		"returnType":       pf.GoType(),
		"originName":       ms.getOriginName(),
		"originStructName": ms.getOriginFullName(),
		"originStructType": ms.getOriginFullName() + "_" + opv.fieldName,
	}
}

var _ Field = (*OptionalPrimitiveField)(nil)
