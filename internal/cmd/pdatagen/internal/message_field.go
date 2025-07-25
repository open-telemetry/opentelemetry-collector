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
	return {{ .packageName }}{{ .returnType }}(internal.New{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .fieldName }}, ms.state))
	{{- else }}
	return new{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .fieldName }}, ms.state)
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

const messageSetTestTemplate = `{{ if .isCommon -}}
	{{ if not .isBaseStructCommon }}internal.{{ end }}FillTest{{ .returnType }}(
	{{- if not .isBaseStructCommon }}internal.{{ end }}New
	{{- else -}}
	fillTest{{ .returnType }}(new
	{{-	end -}}
	{{ .returnType }}(&tv.orig.{{ .originFieldName }}, tv.state))`

const messageCopyOrigTemplate = `{{ if .isCommon }}{{ if not .isBaseStructCommon }}internal.{{ end }}CopyOrig{{ else }}copyOrig{{ end }}
{{- .returnType }}(&dest.{{ .originFieldName }}, &src.{{ .originFieldName }})`

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
