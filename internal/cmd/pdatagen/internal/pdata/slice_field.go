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

func (sf *SliceField) GenerateTestFailingUnmarshalProtoValues(*messageStruct) string {
	return sf.toProtoField().GenTestFailingUnmarshalProtoValues()
}

func (sf *SliceField) GenerateTestEncodingValues(*messageStruct) string {
	return sf.toProtoField().GenTestEncodingValues()
}

func (sf *SliceField) GeneratePoolOrig(*messageStruct) string {
	return ""
}

func (sf *SliceField) GenerateDeleteOrig(*messageStruct) string {
	return sf.toProtoField().GenDeleteOrig()
}

func (sf *SliceField) GenerateCopyOrig(ms *messageStruct) string {
	t := template.Parse("sliceCopyOrigTemplate", []byte(sliceCopyOrigTemplate))
	return template.Execute(t, sf.templateFields(ms))
}

func (sf *SliceField) GenerateMarshalJSON(*messageStruct) string {
	return sf.toProtoField().GenMarshalJSON()
}

func (sf *SliceField) GenerateUnmarshalJSON(*messageStruct) string {
	return sf.toProtoField().GenUnmarshalJSON()
}

func (sf *SliceField) GenerateSizeProto(*messageStruct) string {
	return sf.toProtoField().GenSizeProto()
}

func (sf *SliceField) GenerateMarshalProto(*messageStruct) string {
	return sf.toProtoField().GenMarshalProto()
}

func (sf *SliceField) GenerateUnmarshalProto(*messageStruct) string {
	return sf.toProtoField().GenUnmarshalProto()
}

func (sf *SliceField) toProtoField() *proto.Field {
	return &proto.Field{
		Type:            sf.protoType,
		ID:              sf.protoID,
		Name:            sf.fieldName,
		MessageFullName: sf.returnSlice.getOriginFullName(),
		Repeated:        sf.protoType != proto.TypeBytes,
		Nullable:        sf.returnSlice.getElementNullable(),
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
