// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"text/template"
)

const sliceAccessorTemplate = `// {{ .fieldName }} returns the {{ .fieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .packageName }}{{ .returnType }} {
	{{- if .elementHasWrapper }}
	return {{ .packageName }}{{ .returnType }}(internal.New{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .originFieldName }}, ms.{{ .stateAccessor }}))
	{{- else }}
	return new{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .originFieldName }}, ms.{{ .stateAccessor }})
	{{- end }}
}`

const sliceAccessorsTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	assert.Equal(t, {{ .packageName }}New{{ .returnType }}(), ms.{{ .fieldName }}())
	ms.{{ .origAccessor }}.{{ .originFieldName }} = internal.GenerateOrigTest{{ .elementOriginName }}Slice()
	{{- if .elementHasWrapper }}
	assert.Equal(t, {{ .packageName }}{{ .returnType }}(internal.GenerateTest{{ .returnType }}()), ms.{{ .fieldName }}())
	{{- else }}
	assert.Equal(t, generateTest{{ .returnType }}(), ms.{{ .fieldName }}())
	{{- end }}
}`

const sliceSetTestTemplate = `orig.{{ .originFieldName }} = GenerateOrigTest{{ .elementOriginName }}Slice()`

const sliceCopyOrigTemplate = `dest.{{ .originFieldName }} = CopyOrig{{ .elementOriginName }}Slice(dest.{{ .originFieldName }}, src.{{ .originFieldName }})`

const sliceUnmarshalJSONTemplate = `case "{{ lowerFirst .originFieldName }}"{{ if needSnake .originFieldName -}}, "{{ toSnake .originFieldName }}"{{- end }}:
	orig.{{ .originFieldName }} = UnmarshalJSONOrig{{ .elementOriginName }}Slice(iter)`

type SliceField struct {
	fieldName     string
	protoType     ProtoType
	protoID       uint32
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

func (sf *SliceField) GenerateTestValue(*messageStruct) string { return "" }

func (sf *SliceField) GenerateCopyOrig(ms *messageStruct) string {
	t := template.Must(templateNew("sliceCopyOrigTemplate").Parse(sliceCopyOrigTemplate))
	return executeTemplate(t, sf.templateFields(ms))
}

func (sf *SliceField) GenerateMarshalJSON(*messageStruct) string {
	return sf.toProtoField().genMarshalJSON()
}

func (sf *SliceField) GenerateUnmarshalJSON(ms *messageStruct) string {
	t := template.Must(templateNew("sliceUnmarshalJSONTemplate").Parse(sliceUnmarshalJSONTemplate))
	return executeTemplate(t, sf.templateFields(ms))
}

func (sf *SliceField) GenerateSizeProto(*messageStruct) string {
	return sf.toProtoField().genSizeProto()
}

func (sf *SliceField) GenerateMarshalProto(*messageStruct) string {
	return sf.toProtoField().genMarshalProto()
}

func (sf *SliceField) toProtoField() *ProtoField {
	_, nullable := sf.returnSlice.(*sliceOfPtrs)
	return &ProtoField{
		Type:        sf.protoType,
		ID:          sf.protoID,
		Name:        sf.fieldName,
		MessageName: sf.returnSlice.getElementOriginName(),
		Repeated:    sf.protoType != ProtoTypeBytes,
		Nullable:    nullable,
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
	}
}

var _ Field = (*SliceField)(nil)
