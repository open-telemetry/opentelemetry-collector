// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const sliceAccessorTemplate = `// {{ .fieldName }} returns the {{ .fieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .packageName }}{{ .returnType }} {
	{{- if .elementHasWrapper }}
	return {{ .packageName }}{{ .returnType }}(internal.New{{ .returnType }}Wrapper(&ms.{{ .origAccessor }}.{{ .originFieldName }}, ms.{{ .stateAccessor }}))
	{{- else }}
	return new{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .originFieldName }}, ms.{{ .stateAccessor }})
	{{- end }}
}`

const sliceAccessorsTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	assert.Equal(t, {{ .packageName }}New{{ .returnType }}(), ms.{{ .fieldName }}())
	ms.{{ .origAccessor }}.{{ .originFieldName }} = internal.GenTest{{ .elementOriginName }}{{ if .elementNullable }}Ptr{{ end }}Slice()
	{{- if .elementHasWrapper }}
	assert.Equal(t, {{ .packageName }}{{ .returnType }}(internal.GenTest{{ .returnType }}Wrapper()), ms.{{ .fieldName }}())
	{{- else }}
	assert.Equal(t, generateTest{{ .returnType }}(), ms.{{ .fieldName }}())
	{{- end }}
}`

const sliceSetTestTemplate = `orig.{{ .originFieldName }} = internal.GenTest{{ .elementOriginName }}{{ if .elementNullable }}Ptr{{ end }}Slice()`

type SliceField struct {
	fieldName     string
	protoType     proto.Type
	protoID       uint32
	returnSlice   baseSlice
	hideAccessors bool
}

func (sf *SliceField) GenerateAccessors(ms *messageStruct) string {
	if sf.hideAccessors {
		return ""
	}
	t := template.Parse("sliceAccessorTemplate", []byte(sliceAccessorTemplate))
	return template.Execute(t, sf.templateFields(ms))
}

func (sf *SliceField) GenerateAccessorsTest(ms *messageStruct) string {
	if sf.hideAccessors {
		return ""
	}
	t := template.Parse("sliceAccessorsTestTemplate", []byte(sliceAccessorsTestTemplate))
	return template.Execute(t, sf.templateFields(ms))
}

func (sf *SliceField) GenerateTestValue(ms *messageStruct) string {
	t := template.Parse("sliceSetTestTemplate", []byte(sliceSetTestTemplate))
	return template.Execute(t, sf.templateFields(ms))
}

func (sf *SliceField) toProtoField(ms *messageStruct) proto.FieldInterface {
	return &proto.Field{
		Type:              sf.protoType,
		ID:                sf.protoID,
		Name:              sf.fieldName,
		MessageName:       sf.returnSlice.getElementOriginName(),
		ParentMessageName: ms.protoName,
		Repeated:          sf.protoType != proto.TypeBytes,
		Nullable:          sf.returnSlice.getElementNullable(),
	}
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
		"returnType":        sf.returnSlice.getName(),
		"origAccessor":      origAccessor(ms.getHasWrapper()),
		"stateAccessor":     stateAccessor(ms.getHasWrapper()),
		"elementHasWrapper": sf.returnSlice.getHasWrapper(),
		"elementNullable":   sf.returnSlice.getElementNullable(),
	}
}

var _ Field = (*SliceField)(nil)
