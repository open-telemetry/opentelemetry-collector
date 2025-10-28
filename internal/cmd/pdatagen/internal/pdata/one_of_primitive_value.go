// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	"strings"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

const oneOfPrimitiveAccessorsTemplate = `// {{ .accessorFieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .accessorFieldName }}() {{ .returnType }} {
	return ms.orig.Get{{ .originFieldName }}()
}

// Set{{ .accessorFieldName }} replaces the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) Set{{ .accessorFieldName }}(v {{ .returnType }}) {
	ms.state.AssertMutable()
	var ov *internal.{{ .originStructType }}
	if !internal.UseProtoPooling.IsEnabled() {
		ov = &internal.{{ .originStructType }}{}
	} else {
		ov = internal.ProtoPool{{ .oneOfName }}.Get().(*internal.{{ .originStructType }})
	}
	ov.{{ .originFieldName }} = v
	ms.orig.{{ .originOneOfFieldName }} = ov
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
	sharedState := internal.NewState()
	sharedState.MarkReadOnly()
	assert.Panics(t, func() { new{{ .structName }}(internal.New{{ .originStructName }}(), sharedState).Set{{ .accessorFieldName }}({{ .testValue }}) })
}
`

const oneOfPrimitiveSetTestTemplate = `orig.{{ .originOneOfFieldName }} = &internal.{{ .originStructType }}{
{{- .originFieldName }}: {{ .testValue }}}`

const oneOfPrimitiveTypeTemplate = `case *internal.{{ .originStructType }}:
	return {{ .typeName }}`

type OneOfPrimitiveValue struct {
	fieldName       string
	protoID         uint32
	protoType       proto.Type
	originFieldName string
}

func (opv *OneOfPrimitiveValue) GetOriginFieldName() string {
	return opv.originFieldName
}

func (opv *OneOfPrimitiveValue) GenerateAccessors(ms *messageStruct, of *OneOfField) string {
	t := template.Parse("oneOfPrimitiveAccessorsTemplate", []byte(oneOfPrimitiveAccessorsTemplate))
	return template.Execute(t, opv.templateFields(ms, of))
}

func (opv *OneOfPrimitiveValue) GenerateTests(ms *messageStruct, of *OneOfField) string {
	t := template.Parse("oneOfPrimitiveAccessorTestTemplate", []byte(oneOfPrimitiveAccessorTestTemplate))
	return template.Execute(t, opv.templateFields(ms, of))
}

func (opv *OneOfPrimitiveValue) GenerateTestValue(ms *messageStruct, of *OneOfField) string {
	t := template.Parse("oneOfPrimitiveSetTestTemplate", []byte(oneOfPrimitiveSetTestTemplate))
	return template.Execute(t, opv.templateFields(ms, of))
}

func (opv *OneOfPrimitiveValue) GenerateType(ms *messageStruct, of *OneOfField) string {
	t := template.Parse("oneOfPrimitiveCopyOrigTemplate", []byte(oneOfPrimitiveTypeTemplate))
	return template.Execute(t, opv.templateFields(ms, of))
}

func (opv *OneOfPrimitiveValue) toProtoField(ms *messageStruct, of *OneOfField) proto.FieldInterface {
	pf := &proto.Field{
		Type:              opv.protoType,
		ID:                opv.protoID,
		OneOfGroup:        of.originFieldName,
		Name:              opv.originFieldName,
		OneOfMessageName:  ms.protoName + "_" + opv.originFieldName,
		ParentMessageName: ms.protoName,
		Nullable:          true,
	}
	return pf
}

func (opv *OneOfPrimitiveValue) templateFields(ms *messageStruct, of *OneOfField) map[string]any {
	pf := opv.toProtoField(ms, of)
	return map[string]any{
		"structName":              ms.getName(),
		"defaultVal":              pf.DefaultValue(),
		"packageName":             "",
		"accessorFieldName":       opv.getAccessorFieldName(of),
		"testValue":               pf.TestValue(),
		"originOneOfTypeFuncName": of.typeFuncName(),
		"typeName":                of.typeName + opv.fieldName,
		"lowerFieldName":          strings.ToLower(opv.fieldName),
		"returnType":              pf.GoType(),
		"originFieldName":         opv.originFieldName,
		"originOneOfFieldName":    of.originFieldName,
		"originStructName":        ms.protoName,
		"originStructType":        ms.protoName + "_" + opv.originFieldName,
		"oneOfName":               proto.ExtractNameFromFull(ms.protoName + "_" + opv.originFieldName),
	}
}

func (opv *OneOfPrimitiveValue) getAccessorFieldName(of *OneOfField) string {
	if of.omitOriginFieldNameInNames {
		return opv.fieldName
	}
	return opv.fieldName + of.originFieldName
}

var _ oneOfValue = (*OneOfPrimitiveValue)(nil)
