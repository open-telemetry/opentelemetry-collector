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
	return {{ .typeName }}(ms.{{ .origAccessor }}.{{ .originFieldName }}Type())
}

{{ range .values -}}
{{ .GenerateAccessors $.baseStruct $.OneOfField }}
{{- end }}`

const oneOfAccessorTestTemplate = `func Test{{ .structName }}_{{ .typeFuncName }}(t *testing.T) {
	tv := New{{ .structName }}()
	assert.Equal(t, {{ .typeName }}Empty, tv.{{ .typeFuncName }}())
}

{{ range .values -}}
{{ .GenerateTests $.baseStruct $.OneOfField }}
{{- end }}
`

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
	fields := make([]*proto.Field, len(of.values))
	for i := range of.values {
		fields[i] = of.values[i].toProtoField(ms, of).(*proto.Field)
	}
	return &proto.OneOfField{
		GroupName: of.originFieldName,
		Name:      ms.protoName,
		Fields:    fields,
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
