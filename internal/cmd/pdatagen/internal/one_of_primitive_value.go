// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"strings"
	"text/template"
)

const oneOfPrimitiveAccessorsTemplate = `// {{ .accessorFieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .accessorFieldName }}() {{ .returnType }} {
	return ms.orig.Get{{ .originFieldName }}()
}

// Set{{ .accessorFieldName }} replaces the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) Set{{ .accessorFieldName }}(v {{ .returnType }}) {
	ms.state.AssertMutable()
	ms.orig.{{ .originOneOfFieldName }} = &{{ .originStructType }}{
		{{ .originFieldName }}: v,
	}
}`

const oneOfPrimitiveAccessorTestTemplate = `func Test{{ .structName }}_{{ .accessorFieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	{{- if eq .returnType "float64"}}
	assert.InDelta(t, {{ .defaultVal }}, ms.{{ .accessorFieldName }}(), 0.01)
	{{- else if and (eq .returnType "string") (eq .defaultVal "\"\"") }}
	assert.Empty(t, ms.{{ .accessorFieldName }}())
	{{- else }}
	assert.Equal(t, {{ .defaultVal }}, ms.{{ .accessorFieldName }}())
	{{- end }}
	ms.Set{{ .accessorFieldName }}({{ .testValue }})
	{{- if eq .returnType "float64" }}
	assert.InDelta(t, {{ .testValue }}, ms.{{ .accessorFieldName }}(), 0.01)
	{{- else if and (eq .returnType "string") (eq .testValue "\"\"") }}
	assert.Empty(t, ms.{{ .accessorFieldName }}())
	{{- else }}
	assert.Equal(t, {{ .testValue }}, ms.{{ .accessorFieldName }}())
	{{- end }}
	assert.Equal(t, {{ .typeName }}, ms.{{ .originOneOfTypeFuncName }}())
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { new{{ .structName }}(&{{ .originStructName }}{}, &sharedState).Set{{ .accessorFieldName }}({{ .testValue }}) })
}
`

const oneOfPrimitiveSetTestTemplate = `tv.orig.{{ .originOneOfFieldName }} = &{{ .originStructName }}_{{ .originFieldName }}{
{{- .originFieldName }}: {{ .testValue }}}`

const oneOfPrimitiveCopyOrigTemplate = `case *{{ .originStructName }}_{{ .originFieldName }}:
	dest.{{ .originOneOfFieldName }} = &{{ .originStructName }}_{{ .originFieldName }}{
{{- .originFieldName }}: t.{{ .originFieldName }}}`

const oneOfPrimitiveTypeTemplate = `case *{{ .originStructName }}_{{ .originFieldName }}:
	return {{ .typeName }}`

const oneOfPrimitiveMarshalJSONTemplate = `case *{{ .originStructName }}_{{ .originFieldName }}:
	dest.WriteObjectField("{{ lowerFirst .originFieldName }}")
	dest.Write{{ upperFirst .returnType }}(ov.{{ .originFieldName }})`

const oneOfPrimitiveUnmarshalJSONTemplate = `case "{{ lowerFirst .originFieldName }}"{{ if needSnake .originFieldName -}}, "{{ toSnake .originFieldName }}"{{- end }}:
	ms.orig.{{ .originOneOfFieldName }} = &{{ .originStructType }}{
		{{ .originFieldName }}: iter.Read{{ upperFirst .returnType }}(),
	}`

type OneOfPrimitiveValue struct {
	fieldName       string
	defaultVal      string
	testVal         string
	returnType      string
	originFieldName string
}

func (opv *OneOfPrimitiveValue) GenerateAccessors(ms *messageStruct, of *OneOfField) string {
	t := template.Must(templateNew("oneOfPrimitiveAccessorsTemplate").Parse(oneOfPrimitiveAccessorsTemplate))
	return executeTemplate(t, opv.templateFields(ms, of))
}

func (opv *OneOfPrimitiveValue) GenerateTests(ms *messageStruct, of *OneOfField) string {
	t := template.Must(templateNew("oneOfPrimitiveAccessorTestTemplate").Parse(oneOfPrimitiveAccessorTestTemplate))
	return executeTemplate(t, opv.templateFields(ms, of))
}

func (opv *OneOfPrimitiveValue) GenerateSetWithTestValue(ms *messageStruct, of *OneOfField) string {
	t := template.Must(templateNew("oneOfPrimitiveSetTestTemplate").Parse(oneOfPrimitiveSetTestTemplate))
	return executeTemplate(t, opv.templateFields(ms, of))
}

func (opv *OneOfPrimitiveValue) GenerateCopyOrig(ms *messageStruct, of *OneOfField) string {
	t := template.Must(templateNew("oneOfPrimitiveCopyOrigTemplate").Parse(oneOfPrimitiveCopyOrigTemplate))
	return executeTemplate(t, opv.templateFields(ms, of))
}

func (opv *OneOfPrimitiveValue) GenerateType(ms *messageStruct, of *OneOfField) string {
	t := template.Must(templateNew("oneOfPrimitiveCopyOrigTemplate").Parse(oneOfPrimitiveTypeTemplate))
	return executeTemplate(t, opv.templateFields(ms, of))
}

func (opv *OneOfPrimitiveValue) GenerateMarshalJSON(ms *messageStruct, of *OneOfField) string {
	t := template.Must(templateNew("oneOfPrimitiveMarshalJSONTemplate").Parse(oneOfPrimitiveMarshalJSONTemplate))
	return executeTemplate(t, opv.templateFields(ms, of))
}

func (opv *OneOfPrimitiveValue) GenerateUnmarshalJSON(ms *messageStruct, of *OneOfField) string {
	t := template.Must(templateNew("oneOfPrimitiveUnmarshalJSONTemplate").Parse(oneOfPrimitiveUnmarshalJSONTemplate))
	return executeTemplate(t, opv.templateFields(ms, of))
}

func (opv *OneOfPrimitiveValue) templateFields(ms *messageStruct, of *OneOfField) map[string]any {
	return map[string]any{
		"structName":  ms.getName(),
		"defaultVal":  opv.defaultVal,
		"packageName": "",
		"accessorFieldName": func() string {
			if of.omitOriginFieldNameInNames {
				return opv.fieldName
			}
			return opv.fieldName + of.originFieldName
		}(),
		"testValue":               opv.testVal,
		"originOneOfTypeFuncName": of.typeFuncName(),
		"typeName":                of.typeName + opv.fieldName,
		"lowerFieldName":          strings.ToLower(opv.fieldName),
		"returnType":              opv.returnType,
		"originFieldName":         opv.originFieldName,
		"originOneOfFieldName":    of.originFieldName,
		"originStructName":        ms.originFullName,
		"originStructType":        ms.originFullName + "_" + opv.originFieldName,
	}
}

var _ oneOfValue = (*OneOfPrimitiveValue)(nil)
