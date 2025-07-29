// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"text/template"
)

const sliceAccessorTemplate = `// {{ .fieldName }} returns the {{ .originFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .packageName }}{{ .returnType }} {
	{{- if .isCommon }}
	return {{ .packageName }}{{ .returnType }}(internal.New{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .originFieldName }}
	{{- if .isBaseStructCommon -}}
	, internal.Get{{ .structName }}State(internal.{{ .structName }}(ms))
	{{- else -}}
	, ms.state
	{{- end -}}
	))
	{{- else }}
	return new{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .originFieldName }}, ms.state)
	{{- end }}
}`

const sliceAccessorsTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	assert.Equal(t, {{ .packageName }}New{{ .returnType }}(), ms.{{ .fieldName }}())
	{{- if .isCommon }}
	internal.FillTest{{ .returnType }}(internal.{{ .returnType }}(ms.{{ .fieldName }}()))
	assert.Equal(t, {{ .packageName }}{{ .returnType }}(internal.GenerateTest{{ .returnType }}()), ms.{{ .fieldName }}())
	{{- else }}
	fillTest{{ .returnType }}(ms.{{ .fieldName }}())
	assert.Equal(t, generateTest{{ .returnType }}(), ms.{{ .fieldName }}())
	{{- end }}
}`

const sliceSetTestTemplate = `{{ if .isCommon -}}
	{{ if not .isBaseStructCommon }}internal.{{ end }}FillTest{{ .returnType }}(
	{{- if not .isBaseStructCommon }}internal.{{ end }}New
	{{- else -}}
	fillTest{{ .returnType }}(new
	{{-	end -}}
	{{ .returnType }}(&tv.orig.{{ .originFieldName }}, tv.state))`

const sliceCopyOrigTemplate = `dest.{{ .originFieldName }} = 
{{- if .isCommon }}{{ if not .isBaseStructCommon }}internal.{{ end }}CopyOrig{{ else }}copyOrig{{ end }}
{{- .returnType }}(dest.{{ .originFieldName }}, src.{{ .originFieldName }})`

const sliceMarshalJSONTemplate = `if len(ms.orig.{{ .originFieldName }}) > 0 {
		dest.WriteObjectField("{{ lowerFirst .originFieldName }}")
		{{- if .isCommon }}
		{{ if not .isBaseStructCommon }}internal.{{ end }}MarshalJSONStream{{ .returnType }}(
		{{- if not .isBaseStructCommon }}internal.{{ end }}New{{ .returnType }}(&ms.orig.{{ .originFieldName }}, ms.state), dest)
		{{- else }}
		ms.{{ .fieldName }}().marshalJSONStream(dest)
		{{- end }}
	}`

const sliceUnmarshalJSONTemplate = `case "{{ lowerFirst .originFieldName }}"{{ if needSnake .originFieldName -}}, "{{ toSnake .originFieldName }}"{{- end }}:
	{{- if .isCommon }}
	{{ if not .isBaseStructCommon }}internal.{{ end }}UnmarshalJSONIter{{ .returnType }}(
	{{- if not .isBaseStructCommon }}internal.{{ end }}New{{ .returnType }}(&ms.orig.{{ .originFieldName }}, ms.state), iter)
	{{- else }}
	ms.{{ .fieldName }}().unmarshalJSONIter(iter)
	{{- end }}`

type SliceField struct {
	fieldName       string
	originFieldName string
	returnSlice     baseSlice
	hideAccessors   bool
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
		"structName": ms.getName(),
		"fieldName":  sf.fieldName,
		"packageName": func() string {
			if sf.returnSlice.getPackageName() != ms.packageName {
				return sf.returnSlice.getPackageName() + "."
			}
			return ""
		}(),
		"returnType":         sf.returnSlice.getName(),
		"origAccessor":       origAccessor(ms.packageName),
		"stateAccessor":      stateAccessor(ms.packageName),
		"isCommon":           usedByOtherDataTypes(sf.returnSlice.getPackageName()),
		"isBaseStructCommon": usedByOtherDataTypes(ms.packageName),
		"originFieldName": func() string {
			if sf.originFieldName == "" {
				return sf.fieldName
			}
			return sf.originFieldName
		}(),
	}
}

var _ Field = (*SliceField)(nil)
