// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"strings"
	"text/template"
)

const messageAccessorsTemplate = `// {{ .fieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .packageName }}{{ .returnType }} {
	{{- if .messageHasWrapper }}
	return {{ .packageName }}{{ .returnType }}(internal.New{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .fieldOriginFullName }}, ms.{{ .stateAccessor }}))
	{{- else }}
	return new{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .fieldOriginFullName }}, ms.{{ .stateAccessor }})
	{{- end }}
}`

const messageAccessorsTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	assert.Equal(t, {{ .packageName }}New{{ .returnType }}{{- if eq .returnType "Value" }}Empty{{- end }}(), ms.{{ .fieldName }}())
	internal.FillOrigTest{{ .fieldOriginName }}(&ms.{{ .origAccessor }}.{{ .fieldOriginFullName }})
	{{- if .messageHasWrapper }}
	assert.Equal(t, {{ .packageName }}{{ .returnType }}(internal.GenerateTest{{ .returnType }}()), ms.{{ .fieldName }}())
	{{- else }}
	assert.Equal(t, generateTest{{ .returnType }}(), ms.{{ .fieldName }}())
	{{- end }}
}`

const messageSetTestTemplate = `FillOrigTest{{ .fieldOriginName }}(&orig.{{ .fieldOriginFullName }})`

const messageCopyOrigTemplate = `CopyOrig{{ .fieldOriginName }}(&dest.{{ .fieldOriginFullName }}, &src.{{ .fieldOriginFullName }})`

const messageUnmarshalJSONTemplate = `case "{{ lowerFirst .fieldOriginFullName }}"{{ if needSnake .fieldOriginFullName -}}, "{{ toSnake .fieldOriginFullName }}"{{- end }}:
	UnmarshalJSONOrig{{ .fieldOriginName }}(&orig.{{ .fieldOriginFullName }}, iter)`

type MessageField struct {
	fieldName     string
	protoID       uint32
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

func (mf *MessageField) GenerateTestValue(*messageStruct) string { return "" }

func (mf *MessageField) GenerateCopyOrig(ms *messageStruct) string {
	t := template.Must(templateNew("messageCopyOrigTemplate").Parse(messageCopyOrigTemplate))
	return executeTemplate(t, mf.templateFields(ms))
}

func (mf *MessageField) GenerateMarshalJSON(*messageStruct) string {
	return mf.toProtoField().genMarshalJSON()
}

func (mf *MessageField) GenerateUnmarshalJSON(ms *messageStruct) string {
	t := template.Must(templateNew("messageUnmarshalJSONTemplate").Parse(messageUnmarshalJSONTemplate))
	return executeTemplate(t, mf.templateFields(ms))
}

func (mf *MessageField) GenerateSizeProto(*messageStruct) string {
	return mf.toProtoField().genSizeProto()
}

func (mf *MessageField) GenerateMarshalProto(*messageStruct) string {
	return mf.toProtoField().genMarshalProto()
}

func (mf *MessageField) GenerateUnmarshalProto(*messageStruct) string {
	return mf.toProtoField().genUnmarshalProto()
}

func (mf *MessageField) toProtoField() *ProtoField {
	pt := ProtoTypeMessage
	if mf.returnMessage.getName() == "TraceState" {
		pt = ProtoTypeString
	}
	return &ProtoField{
		Type:            pt,
		ID:              mf.protoID,
		Name:            mf.fieldName,
		MessageFullName: mf.returnMessage.getOriginFullName(),
	}
}

func (mf *MessageField) templateFields(ms *messageStruct) map[string]any {
	return map[string]any{
		"messageHasWrapper":   usedByOtherDataTypes(mf.returnMessage.packageName),
		"structName":          ms.getName(),
		"fieldName":           mf.fieldName,
		"fieldOriginFullName": mf.fieldName,
		"fieldOriginName":     mf.returnMessage.getOriginName(),
		"lowerFieldName":      strings.ToLower(mf.fieldName),
		"returnType":          mf.returnMessage.getName(),
		"packageName": func() string {
			if mf.returnMessage.packageName != ms.packageName {
				return mf.returnMessage.packageName + "."
			}
			return ""
		}(),
		"origAccessor":  origAccessor(ms.getHasWrapper()),
		"stateAccessor": stateAccessor(ms.getHasWrapper()),
	}
}

var _ Field = (*MessageField)(nil)
