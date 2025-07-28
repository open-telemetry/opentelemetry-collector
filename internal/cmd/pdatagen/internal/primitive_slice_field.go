// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"strings"
	"text/template"
)

const primitiveSliceAccessorsTemplate = `// {{ .fieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .packageName }}{{ .returnType }} {
	return {{ .packageName }}{{ .returnType }}(internal.New{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .fieldName }}, ms.state))
}`

const primitiveSliceAccessorsTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	assert.Equal(t, {{ .defaultVal }}, ms.{{ .fieldName }}().AsRaw())
	ms.{{ .fieldName }}().FromRaw({{ .testValue }})
	assert.Equal(t, {{ .testValue }}, ms.{{ .fieldName }}().AsRaw())
}`

const primitiveSliceSetTestTemplate = `tv.orig.{{ .originFieldName }} = {{ .testValue }}`

const primitiveSliceCopyOrigTemplate = `dest.{{ .originFieldName }} = 
{{- if .isCommon }}{{ if not .isBaseStructCommon }}internal.{{ end }}CopyOrig{{ else }}copyOrig{{ end }}
{{- .returnType }}(dest.{{ .originFieldName }}, src.{{ .originFieldName }})`

const primitiveSliceMarshalJSONTemplate = `dest.WriteObjectField("{{ lowerFirst .originFieldName }}")
	{{- if .isCommon }}
	{{ if not .isBaseStructCommon }}internal.{{ end }}MarshalJSONStream{{ .returnType }}(
	{{- if not .isBaseStructCommon }}internal.{{ end }}New{{ .returnType }}(&ms.orig.{{ .originFieldName }}, ms.state), dest)
	{{- else }}
	ms.{{ .fieldName }}().marshalJSONStream(dest)
	{{- end }}`

// PrimitiveSliceField is used to generate fields for slice of primitive types
type PrimitiveSliceField struct {
	fieldName         string
	returnPackageName string
	returnType        string
	defaultVal        string
	rawType           string
	testVal           string
}

func (psf *PrimitiveSliceField) GenerateAccessors(ms *messageStruct) string {
	t := template.Must(templateNew("primitiveSliceAccessorsTemplate").Parse(primitiveSliceAccessorsTemplate))
	return executeTemplate(t, psf.templateFields(ms))
}

func (psf *PrimitiveSliceField) GenerateAccessorsTest(ms *messageStruct) string {
	t := template.Must(templateNew("primitiveSliceAccessorsTestTemplate").Parse(primitiveSliceAccessorsTestTemplate))
	return executeTemplate(t, psf.templateFields(ms))
}

func (psf *PrimitiveSliceField) GenerateSetWithTestValue(ms *messageStruct) string {
	t := template.Must(templateNew("primitiveSliceSetTestTemplate").Parse(primitiveSliceSetTestTemplate))
	return executeTemplate(t, psf.templateFields(ms))
}

func (psf *PrimitiveSliceField) GenerateCopyOrig(ms *messageStruct) string {
	t := template.Must(templateNew("primitiveSliceCopyOrigTemplate").Parse(primitiveSliceCopyOrigTemplate))
	return executeTemplate(t, psf.templateFields(ms))
}

func (psf *PrimitiveSliceField) GenerateMarshalJSON(ms *messageStruct) string {
	t := template.Must(templateNew("primitiveSliceMarshalJSONTemplate").Parse(primitiveSliceMarshalJSONTemplate))
	return executeTemplate(t, psf.templateFields(ms))
}

func (psf *PrimitiveSliceField) templateFields(ms *messageStruct) map[string]any {
	return map[string]any{
		"structName": ms.getName(),
		"packageName": func() string {
			if psf.returnPackageName != ms.packageName {
				return psf.returnPackageName + "."
			}
			return ""
		}(),
		"isCommon":           usedByOtherDataTypes(psf.returnPackageName),
		"isBaseStructCommon": usedByOtherDataTypes(ms.packageName),
		"returnType":         psf.returnType,
		"defaultVal":         psf.defaultVal,
		"fieldName":          psf.fieldName,
		"originFieldName":    psf.fieldName,
		"lowerFieldName":     strings.ToLower(psf.fieldName),
		"testValue":          psf.testVal,
		"origAccessor":       origAccessor(ms.packageName),
		"stateAccessor":      stateAccessor(ms.packageName),
	}
}

var _ Field = (*PrimitiveSliceField)(nil)
