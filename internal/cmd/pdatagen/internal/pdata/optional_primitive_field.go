// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"
import (
	"strings"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/tmplutil"
)

const optionalPrimitiveAccessorsTemplate = `// {{ .fieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .returnType }} {
	return ms.orig.{{ .fieldName }}
}

// Has{{ .fieldName }} returns true if the {{ .structName }} contains a
// {{ .fieldName }} value otherwise.
func (ms {{ .structName }}) Has{{ .fieldName }}() bool {
	return ms.orig.Has{{ .fieldName }}()
}

// Set{{ .fieldName }} replaces the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) Set{{ .fieldName }}(v {{ .returnType }}) {
	ms.state.AssertMutable()
	ms.orig.Set{{ .fieldName }}(v)
}

// Remove{{ .fieldName }} removes the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) Remove{{ .fieldName }}() {
	ms.state.AssertMutable()
	ms.orig.Remove{{ .fieldName }}()
}`

const optionalPrimitiveAccessorsTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	{{- if eq .returnType "float64" }}
	assert.InDelta(t, {{ .defaultVal }}, ms.{{ .fieldName }}() , 0.01)
	{{- else }}
	assert.Equal(t, {{ .defaultVal }}, ms.{{ .fieldName }}())
	{{- end }}
	ms.Set{{ .fieldName }}({{ .testValue }})
	assert.True(t, ms.Has{{ .fieldName }}())
	{{- if eq .returnType "float64" }}
	assert.InDelta(t, {{.testValue }}, ms.{{ .fieldName }}(), 0.01)
	{{- else }}
	assert.Equal(t, {{ .testValue }}, ms.{{ .fieldName }}())
	{{- end }}
	ms.Remove{{ .fieldName }}()
	assert.False(t, ms.Has{{ .fieldName }}())
	dest := New{{ .structName }}()
	dest.Set{{ .fieldName }}({{ .testValue }})
	ms.CopyTo(dest)
	assert.False(t, dest.Has{{ .fieldName }}())
}`

const optionalPrimitiveSetTestTemplate = `orig.{{ .fieldName }}_ = &internal.{{ .originStructType }}{
{{- .fieldName }}: {{ .testValue }}}`

type OptionalPrimitiveField struct {
	fieldName string
	protoID   uint32
	protoType proto.Type
}

func (opv *OptionalPrimitiveField) GenerateAccessors(ms *messageStruct) string {
	return tmplutil.Execute(tmplutil.Parse("optionalPrimitiveAccessorsTemplate", []byte(optionalPrimitiveAccessorsTemplate)), opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) GenerateAccessorsTest(ms *messageStruct) string {
	return tmplutil.Execute(tmplutil.Parse("optionalPrimitiveAccessorsTestTemplate", []byte(optionalPrimitiveAccessorsTestTemplate)), opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) GenerateTestValue(ms *messageStruct) string {
	return tmplutil.Execute(tmplutil.Parse("optionalPrimitiveSetTestTemplate", []byte(optionalPrimitiveSetTestTemplate)), opv.templateFields(ms))
}

func (opv *OptionalPrimitiveField) toProtoField(ms *messageStruct) proto.FieldInterface {
	return &proto.Field{
		Type:              opv.protoType,
		ID:                opv.protoID,
		Name:              opv.fieldName,
		ParentMessageName: ms.protoName,
		Nullable:          true,
	}
}

func (opv *OptionalPrimitiveField) templateFields(ms *messageStruct) map[string]any {
	pf := opv.toProtoField(ms).(*proto.Field)
	return map[string]any{
		"structName":       ms.getName(),
		"defaultVal":       pf.DefaultValue(),
		"fieldName":        opv.fieldName,
		"lowerFieldName":   strings.ToLower(opv.fieldName),
		"testValue":        pf.TestValue(),
		"returnType":       pf.GoType(),
		"originName":       ms.getOriginName(),
		"originStructName": ms.getOriginFullName(),
		"originStructType": ms.getOriginFullName() + "_" + opv.fieldName,
	}
}

var _ Field = (*OptionalPrimitiveField)(nil)
