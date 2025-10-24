// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	"strings"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const oneOfAccessorTemplate = `// {{ .typeFuncName }} returns the type of the {{ .lowerOriginFieldName }} for this {{ .structName }}.
// Calling this function on zero-initialized {{ .structName }} will cause a panic.
func (ms {{ .structName }}) {{ .typeFuncName }}() {{ .typeName }} {
	switch ms.{{ .origAccessor }}.{{ .originFieldName }}.(type) {
		{{- range .values }}
		{{ .GenerateType $.baseStruct $.OneOfField }}
		{{- end }}
	}
	return {{ .typeName }}Empty
}

{{ range .values }}
{{ .GenerateAccessors $.baseStruct $.OneOfField }}
{{ end }}`

const oneOfAccessorTestTemplate = `func Test{{ .structName }}_{{ .typeFuncName }}(t *testing.T) {
	tv := New{{ .structName }}()
	assert.Equal(t, {{ .typeName }}Empty, tv.{{ .typeFuncName }}())
}

{{ range .values -}}
{{ .GenerateTests $.baseStruct $.OneOfField }}
{{ end }}
`

const oneOfTestFailingUnmarshalProtoValuesTemplate = `
	{{- range .fields }}
	{{ .GenTestFailingUnmarshalProtoValues }}
	{{- end }}`

const oneOfTestValuesTemplate = `
	{{- range .fields }}
	{{ .GenTestEncodingValues }}
	{{- end }}`

const oneOfPoolOrigTemplate = `
	{{- range .fields }}
	{{ .GenPool }}
	{{- end }}`

const oneOfMessageOrigTemplate = `
func (m *{{ .protoName }}) Get{{ .originFieldName }}() any {
	if m != nil {
		return m.{{ .originFieldName }}
	}
	return nil
}

{{- range .fields }}
{{ .GenOneOfMessages }}
{{- end }}`

const oneOfDeleteOrigTemplate = `switch ov := orig.{{ .originFieldName }}.(type) {
	{{ range .fields -}}
	case *{{ $.protoName }}_{{ .GetName }}:
		{{ .GenDelete }}{{- end }}
	}
`

const oneOfCopyOrigTemplate = `switch t := src.{{ .originFieldName }}.(type) {
{{- range .fields }}
	case *{{ $.protoName }}_{{ .GetName }}:
		{{ .GenCopy }}
{{- end }}
	default:
		dest.{{ .originFieldName }} = nil
}`

const oneOfMarshalJSONTemplate = `switch orig := orig.{{ .originFieldName }}.(type) {
	{{- range .fields }}
	case *{{ $.protoName }}_{{ .GetName }}:
		{{ .GenMarshalJSON }}
	{{- end }}
}`

const oneOfUnmarshalJSONTemplate = `
	{{- range .fields }}
	{{ .GenUnmarshalJSON }}
	{{- end }}`

const oneOfSizeProtoTemplate = `switch orig := orig.{{ .originFieldName }}.(type) {
	case nil:
		_ = orig
		break
	{{ range .fields -}}
	case *{{ $.protoName }}_{{ .GetName }}:
		{{ .GenSizeProto }}
	{{ end -}}
}`

const oneOfMarshalProtoTemplate = `switch orig := orig.{{ .originFieldName }}.(type) {
	{{- range .fields }}
	case *{{ $.protoName }}_{{ .GetName }}:
		{{ .GenMarshalProto }}
	{{- end }}
}`

const oneOfUnmarshalProtoTemplate = `
	{{- range .fields }}
		{{ .GenUnmarshalProto }}
	{{- end }}`

type OneOfField struct {
	originFieldName            string
	typeName                   string
	testValueIdx               int
	values                     []oneOfValue
	omitOriginFieldNameInNames bool
}

func (of *OneOfField) GenerateAccessors(ms *messageStruct) string {
	return template.Execute(template.Parse("oneOfAccessorTemplate", []byte(oneOfAccessorTemplate)), of.templateFields(ms))
}

func (of *OneOfField) typeFuncName() string {
	const typeSuffix = "Type"
	if of.omitOriginFieldNameInNames {
		return typeSuffix
	}
	return of.originFieldName + typeSuffix
}

func (of *OneOfField) GenerateAccessorsTest(ms *messageStruct) string {
	return template.Execute(template.Parse("oneOfAccessorTestTemplate", []byte(oneOfAccessorTestTemplate)), of.templateFields(ms))
}

func (of *OneOfField) GenerateTestValue(ms *messageStruct) string {
	return of.values[of.testValueIdx].GenerateTestValue(ms, of)
}

func (of *OneOfField) toProtoField(ms *messageStruct) proto.FieldInterface {
	fields := make([]proto.FieldInterface, len(of.values))
	for i := range of.values {
		fields[i] = of.values[i].toProtoField(ms, of)
	}
	return &oneOfProtoField{
		originFieldName: of.originFieldName,
		protoName:       ms.protoName,
		fields:          fields,
	}
}

type oneOfProtoField struct {
	originFieldName string
	protoName       string
	fields          []proto.FieldInterface
}

func (of *oneOfProtoField) GenMessageField() string {
	return of.originFieldName + " any"
}

func (of *oneOfProtoField) GenOneOfMessages() string {
	return template.Execute(template.Parse("oneOfMessageOrigTemplate", []byte(oneOfMessageOrigTemplate)), of.templateFields())
}

func (of *oneOfProtoField) GetName() string {
	return of.originFieldName
}

func (of *oneOfProtoField) GoType() string {
	panic("implement me")
}

func (of *oneOfProtoField) DefaultValue() string {
	panic("implement me")
}

func (of *oneOfProtoField) TestValue() string {
	return "&" + of.protoName + "_" + of.fields[0].GetName() + "{" + of.fields[0].GetName() + ": " + of.fields[0].TestValue() + "}"
}

func (of *oneOfProtoField) GenTestFailingUnmarshalProtoValues() string {
	return template.Execute(template.Parse("oneOfTestFailingUnmarshalProtoValuesTemplate", []byte(oneOfTestFailingUnmarshalProtoValuesTemplate)), of.templateFields())
}

func (of *oneOfProtoField) GenTestEncodingValues() string {
	return template.Execute(template.Parse("oneOfTestValuesTemplate", []byte(oneOfTestValuesTemplate)), of.templateFields())
}

func (of *oneOfProtoField) GenPool() string {
	return template.Execute(template.Parse("oneOfPoolOrigTemplate", []byte(oneOfPoolOrigTemplate)), of.templateFields())
}

func (of *oneOfProtoField) GenDelete() string {
	return template.Execute(template.Parse("oneOfDeleteOrigTemplate", []byte(oneOfDeleteOrigTemplate)), of.templateFields())
}

func (of *oneOfProtoField) GenCopy() string {
	return template.Execute(template.Parse("oneOfCopyOrigTemplate", []byte(oneOfCopyOrigTemplate)), of.templateFields())
}

func (of *oneOfProtoField) GenMarshalJSON() string {
	return template.Execute(template.Parse("oneOfMarshalJSONTemplate", []byte(oneOfMarshalJSONTemplate)), of.templateFields())
}

func (of *oneOfProtoField) GenUnmarshalJSON() string {
	return template.Execute(template.Parse("oneOfUnmarshalJSONTemplate", []byte(oneOfUnmarshalJSONTemplate)), of.templateFields())
}

func (of *oneOfProtoField) GenSizeProto() string {
	t := template.Parse("oneOfSizeProtoTemplate", []byte(oneOfSizeProtoTemplate))
	return template.Execute(t, of.templateFields())
}

func (of *oneOfProtoField) GenMarshalProto() string {
	t := template.Parse("oneOfMarshalProtoTemplate", []byte(oneOfMarshalProtoTemplate))
	return template.Execute(t, of.templateFields())
}

func (of *oneOfProtoField) GenUnmarshalProto() string {
	t := template.Parse("oneOfUnmarshalProtoTemplate", []byte(oneOfUnmarshalProtoTemplate))
	return template.Execute(t, of.templateFields())
}

func (of *oneOfProtoField) templateFields() map[string]any {
	return map[string]any{
		"originFieldName": of.originFieldName,
		"fields":          of.fields,
		"protoName":       of.protoName,
	}
}

func (of *OneOfField) templateFields(ms *messageStruct) map[string]any {
	return map[string]any{
		"baseStruct":           ms,
		"OneOfField":           of,
		"packageName":          "",
		"structName":           ms.getName(),
		"typeFuncName":         of.typeFuncName(),
		"typeName":             of.typeName,
		"originName":           ms.getOriginName(),
		"originFieldName":      of.originFieldName,
		"lowerOriginFieldName": strings.ToLower(of.originFieldName),
		"origAccessor":         origAccessor(ms.getHasWrapper()),
		"stateAccessor":        stateAccessor(ms.getHasWrapper()),
		"values":               of.values,
	}
}

var _ Field = (*OneOfField)(nil)

type oneOfValue interface {
	GetOriginFieldName() string
	GenerateAccessors(ms *messageStruct, of *OneOfField) string
	GenerateType(ms *messageStruct, of *OneOfField) string
	GenerateTests(ms *messageStruct, of *OneOfField) string
	GenerateTestValue(ms *messageStruct, of *OneOfField) string
	toProtoField(ms *messageStruct, of *OneOfField) proto.FieldInterface
}
