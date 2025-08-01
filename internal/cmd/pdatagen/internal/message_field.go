// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"strings"
	"text/template"
)

const messageAccessorsTemplate = `// {{ .fieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .packageName }}{{ .returnType }} {
	{{- if .isCommon }}
	return {{ .packageName }}{{ .returnType }}(internal.New{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .originFieldName }}, ms.{{ .stateAccessor }}))
	{{- else }}
	return new{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .originFieldName }}, ms.{{ .stateAccessor }})
	{{- end }}
}`

const messageAccessorsTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	{{- if .isCommon }}
	internal.FillTest{{ .returnType }}(internal.{{ .returnType }}(ms.{{ .fieldName }}()))
	assert.Equal(t, {{ .packageName }}{{ .returnType }}(internal.GenerateTest{{ .returnType }}()), ms.{{ .fieldName }}())
	{{- else }}
	fillTest{{ .returnType }}(ms.{{ .fieldName }}())
	assert.Equal(t, generateTest{{ .returnType }}(), ms.{{ .fieldName }}())
	{{- end }}
}`

const messageSetTestTemplate = `{{- if .isCommon -}}
	internal.FillTest{{ .returnType }}(internal.New{{ .returnType }}(&tv.orig.{{ .originFieldName }}, tv.state))
	{{- else -}}
	fillTest{{ .returnType }}(tv.{{ .fieldName }}())
	{{-	end }}`

const messageCopyOrigTemplate = `{{- if .isCommon -}}
	internal.CopyOrig{{- .returnType }}(&dest.{{ .originFieldName }}, &src.{{ .originFieldName }})
	{{- else -}}
	copyOrig{{ .returnType }}(&dest.{{ .originFieldName }}, &src.{{ .originFieldName }})
	{{- end }}`

const messageMarshalJSONTemplate = `{{- if eq .returnType "TraceState" }} if ms.orig.{{ .originFieldName }} != "" { {{ end -}}
	dest.WriteObjectField("{{ lowerFirst .originFieldName }}")
	{{- if .isCommon }}
	internal.MarshalJSONStream{{ .returnType }}(internal.New{{ .returnType }}(&ms.orig.{{ .originFieldName }}, ms.state), dest)
	{{- else }}
	ms.{{ .fieldName }}().marshalJSONStream(dest)
	{{- end }}{{ if eq .returnType "TraceState" -}} } {{- end }}`

const messageUnmarshalJSONTemplate = `case "{{ lowerFirst .originFieldName }}"{{ if needSnake .originFieldName -}}, "{{ toSnake .originFieldName }}"{{- end }}:
	{{- if .isCommon }}
	internal.UnmarshalJSONIter{{ .returnType }}(internal.New{{ .returnType }}(&ms.orig.{{ .originFieldName }}, ms.state), iter)
	{{- else }}
	ms.{{ .fieldName }}().unmarshalJSONIter(iter)
	{{- end }}`

type MessageField struct {
	fieldName     string
	returnMessage *messageStruct
}

func (mf *MessageField) GenerateAccessors(ms *messageStruct) string {
	t := template.Must(templateNew("messageAccessorsTemplate").Parse(messageAccessorsTemplate))
	return executeTemplate(t, mf.templateFields(ms))
}

func (mf *MessageField) GenerateAccessorsTest(ms *messageStruct) string {
	t := template.Must(templateNew("messageAccessorsTestTemplate").Parse(messageAccessorsTestTemplate))
	return executeTemplate(t, mf.templateFields(ms))
}

func (mf *MessageField) GenerateSetWithTestValue(ms *messageStruct) string {
	t := template.Must(templateNew("messageSetTestTemplate").Parse(messageSetTestTemplate))
	return executeTemplate(t, mf.templateFields(ms))
}

func (mf *MessageField) GenerateCopyOrig(ms *messageStruct) string {
	t := template.Must(templateNew("messageCopyOrigTemplate").Parse(messageCopyOrigTemplate))
	return executeTemplate(t, mf.templateFields(ms))
}

func (mf *MessageField) GenerateMarshalJSON(ms *messageStruct) string {
	t := template.Must(templateNew("messageMarshalJSONTemplate").Parse(messageMarshalJSONTemplate))
	return executeTemplate(t, mf.templateFields(ms))
}

func (mf *MessageField) GenerateUnmarshalJSON(ms *messageStruct) string {
	t := template.Must(templateNew("messageUnmarshalJSONTemplate").Parse(messageUnmarshalJSONTemplate))
	return executeTemplate(t, mf.templateFields(ms))
}

func (mf *MessageField) templateFields(ms *messageStruct) map[string]any {
	return map[string]any{
		"isCommon":        usedByOtherDataTypes(mf.returnMessage.packageName),
		"structName":      ms.getName(),
		"fieldName":       mf.fieldName,
		"originFieldName": mf.fieldName,
		"lowerFieldName":  strings.ToLower(mf.fieldName),
		"returnType":      mf.returnMessage.getName(),
		"packageName": func() string {
			if mf.returnMessage.packageName != ms.packageName {
				return mf.returnMessage.packageName + "."
			}
			return ""
		}(),
		"origAccessor":  origAccessor(ms.packageName),
		"stateAccessor": stateAccessor(ms.packageName),
	}
}

var _ Field = (*MessageField)(nil)
