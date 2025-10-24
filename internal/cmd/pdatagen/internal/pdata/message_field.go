// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	"strings"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const messageAccessorsTemplate = `// {{ .fieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .packageName }}{{ .returnType }} {
	{{- if .messageHasWrapper }}
	return {{ .packageName }}{{ .returnType }}(internal.New{{ .returnType }}Wrapper(&ms.{{ .origAccessor }}.{{ .fieldOriginFullName }}, ms.{{ .stateAccessor }}))
	{{- else }}
	return new{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .fieldOriginFullName }}, ms.{{ .stateAccessor }})
	{{- end }}
}`

const messageAccessorsTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	assert.Equal(t, {{ .packageName }}New{{ .returnType }}{{- if eq .returnType "Value" }}Empty{{- end }}(), ms.{{ .fieldName }}())
	ms.{{ .origAccessor }}.{{ .fieldOriginFullName }} = *internal.GenTest{{ .fieldOriginName }}()
	{{- if .messageHasWrapper }}
	assert.Equal(t, {{ .packageName }}{{ .returnType }}(internal.GenTest{{ .returnType }}Wrapper()), ms.{{ .fieldName }}())
	{{- else }}
	assert.Equal(t, generateTest{{ .returnType }}(), ms.{{ .fieldName }}())
	{{- end }}
}`

const messageSetTestTemplate = `orig.{{ .fieldOriginFullName }} = *GenTest{{ .fieldOriginName }}()`

type MessageField struct {
	fieldName     string
	protoID       uint32
	nullable      bool
	returnMessage *messageStruct
}

func (mf *MessageField) GenerateAccessors(ms *messageStruct) string {
	t := template.Parse("messageAccessorsTemplate", []byte(messageAccessorsTemplate))
	return template.Execute(t, mf.templateFields(ms))
}

func (mf *MessageField) GenerateAccessorsTest(ms *messageStruct) string {
	t := template.Parse("messageAccessorsTestTemplate", []byte(messageAccessorsTestTemplate))
	return template.Execute(t, mf.templateFields(ms))
}

func (mf *MessageField) GenerateTestValue(ms *messageStruct) string {
	t := template.Parse("messageSetTestTemplate", []byte(messageSetTestTemplate))
	return template.Execute(t, mf.templateFields(ms))
}

func (mf *MessageField) toProtoField(ms *messageStruct) proto.FieldInterface {
	pt := proto.TypeMessage
	if mf.returnMessage.getName() == "TraceState" {
		pt = proto.TypeString
	}
	return &proto.Field{
		Type:              pt,
		ID:                mf.protoID,
		Name:              mf.fieldName,
		MessageName:       mf.returnMessage.getOriginName(),
		ParentMessageName: ms.protoName,
		Nullable:          mf.nullable,
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
