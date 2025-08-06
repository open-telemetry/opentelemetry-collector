// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"text/template"
)

const sliceAccessorTemplate = `// {{ .fieldName }} returns the {{ .fieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .packageName }}{{ .returnType }} {
	{{- if .isCommon }}
	return {{ .packageName }}{{ .returnType }}(internal.New{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .originFieldName }}, ms.{{ .stateAccessor }}))
	{{- else }}
	return new{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .originFieldName }}, ms.{{ .stateAccessor }})
	{{- end }}
}`

const sliceAccessorsTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	assert.Equal(t, {{ .packageName }}New{{ .returnType }}(), ms.{{ .fieldName }}())
	ms.{{ .origAccessor }}.{{ .originFieldName }} = internal.GenerateOrigTest{{ .elementOriginName }}Slice()
	{{- if .isCommon }}
	assert.Equal(t, {{ .packageName }}{{ .returnType }}(internal.GenerateTest{{ .returnType }}()), ms.{{ .fieldName }}())
	{{- else }}
	assert.Equal(t, generateTest{{ .returnType }}(), ms.{{ .fieldName }}())
	{{- end }}
}`

const sliceSetTestTemplate = `orig.{{ .originFieldName }} = GenerateOrigTest{{ .elementOriginName }}Slice()`

const sliceCopyOrigTemplate = `dest.{{ .originFieldName }} = CopyOrig{{ .elementOriginName }}Slice(dest.{{ .originFieldName }}, src.{{ .originFieldName }})`

const sliceMarshalJSONTemplate = `if len(orig.{{ .originFieldName }}) > 0 {
		dest.WriteObjectField("{{ lowerFirst .originFieldName }}")
		MarshalJSONOrig{{ .elementOriginName }}Slice(orig.{{ .originFieldName }}, dest)
	}`

const sliceUnmarshalJSONTemplate = `case "{{ lowerFirst .originFieldName }}"{{ if needSnake .originFieldName -}}, "{{ toSnake .originFieldName }}"{{- end }}:
	orig.{{ .originFieldName }} = UnmarshalJSONOrig{{ .elementOriginName }}Slice(iter)`

type SliceField struct {
	fieldName     string
	returnSlice   baseSlice
	hideAccessors bool
}

func (sf *SliceField) GenerateAccessors(ms *messageStruct) string {
	if sf.hideAccessors {
		return ""
	}
	t := template.Must(templateNew("sliceAccessorTemplate").Parse(sliceAccessorTemplate))
	return executeTemplate(t, sf.templateFields(ms))
}

func (sf *SliceField) GenerateAccessorsTest(ms *messageStruct) string {
	if sf.hideAccessors {
		return ""
	}
	t := template.Must(templateNew("sliceAccessorsTestTemplate").Parse(sliceAccessorsTestTemplate))
	return executeTemplate(t, sf.templateFields(ms))
}

func (sf *SliceField) GenerateSetWithTestValue(ms *messageStruct) string {
	t := template.Must(templateNew("sliceSetTestTemplate").Parse(sliceSetTestTemplate))
	return executeTemplate(t, sf.templateFields(ms))
}

func (sf *SliceField) GenerateCopyOrig(ms *messageStruct) string {
	t := template.Must(templateNew("sliceCopyOrigTemplate").Parse(sliceCopyOrigTemplate))
	return executeTemplate(t, sf.templateFields(ms))
}

func (sf *SliceField) GenerateMarshalJSON(ms *messageStruct) string {
	t := template.Must(templateNew("sliceMarshalJSONTemplate").Parse(sliceMarshalJSONTemplate))
	return executeTemplate(t, sf.templateFields(ms))
}

func (sf *SliceField) GenerateUnmarshalJSON(ms *messageStruct) string {
	t := template.Must(templateNew("sliceUnmarshalJSONTemplate").Parse(sliceUnmarshalJSONTemplate))
	return executeTemplate(t, sf.templateFields(ms))
}

func (sf *SliceField) templateFields(ms *messageStruct) map[string]any {
	return map[string]any{
		"structName":        ms.getName(),
		"fieldName":         sf.fieldName,
		"originFieldName":   sf.fieldName,
		"elementOriginName": sf.returnSlice.getElementOriginName(),
		"packageName": func() string {
			if sf.returnSlice.getPackageName() != ms.packageName {
				return sf.returnSlice.getPackageName() + "."
			}
			return ""
		}(),
		"returnType":    sf.returnSlice.getName(),
		"origAccessor":  origAccessor(ms.packageName),
		"stateAccessor": stateAccessor(ms.packageName),
		"isCommon":      usedByOtherDataTypes(sf.returnSlice.getPackageName()),
	}
}

var _ Field = (*SliceField)(nil)
