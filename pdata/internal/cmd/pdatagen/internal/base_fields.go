// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

import (
	"strings"
	"text/template"
)

const accessorSliceTemplate = `// {{ .fieldName }} returns the {{ .originFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .packageName }}{{ .returnType }} {
	{{- if .isCommon }}
	return {{ .packageName }}{{ .returnType }}(internal.New{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .originFieldName }}
	{{- if .isBaseStructCommon -}}
	, internal.Get{{ .structName }}State(internal.{{ .structName }}(ms))
	{{- else -}}
	, ms.state
	{{- end -}}
	))
	{{- else }}
	return new{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .originFieldName }}, ms.state)
	{{- end }}
}`

const accessorsSliceTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	assert.Equal(t, {{ .packageName }}New{{ .returnType }}(), ms.{{ .fieldName }}())
	{{- if .isCommon }}
	internal.FillTest{{ .returnType }}(internal.{{ .returnType }}(ms.{{ .fieldName }}()))
	assert.Equal(t, {{ .packageName }}{{ .returnType }}(internal.GenerateTest{{ .returnType }}()), ms.{{ .fieldName }}())
	{{- else }}
	fillTest{{ .returnType }}(ms.{{ .fieldName }}())
	assert.Equal(t, generateTest{{ .returnType }}(), ms.{{ .fieldName }}())
	{{- end }}
}`

const setTestValueTemplate = `{{ if .isCommon -}}
	{{ if not .isBaseStructCommon }}internal.{{ end }}FillTest{{ .returnType }}(
	{{- if not .isBaseStructCommon }}internal.{{ end }}New
	{{- else -}}
	fillTest{{ .returnType }}(new
	{{-	end -}}
	{{ .returnType }}(&tv.orig.{{ .originFieldName }}, tv.state))`

const copyOrigSliceTemplate = `dest.{{ .originFieldName }} = 
{{- if .isCommon }}{{ if not .isBaseStructCommon }}internal.{{ end }}CopyOrig{{ else }}copyOrig{{ end }}
{{- .returnType }}(dest.{{ .originFieldName }}, src.{{ .originFieldName }})`

const copyOrigMessageTemplate = `{{ if .isCommon }}{{ if not .isBaseStructCommon }}internal.{{ end }}CopyOrig{{ else }}copyOrig{{ end }}
{{- .returnType }}(&dest.{{ .originFieldName }}, &src.{{ .originFieldName }})`

const accessorsMessageValueTemplate = `// {{ .fieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .packageName }}{{ .returnType }} {
	{{- if .isCommon }}
	return {{ .packageName }}{{ .returnType }}(internal.New{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .fieldName }}, ms.state))
	{{- else }}
	return new{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .fieldName }}, ms.state)
	{{- end }}
}`

const accessorsMessageValueTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	{{- if .isCommon }}
	internal.FillTest{{ .returnType }}(internal.{{ .returnType }}(ms.{{ .fieldName }}()))
	assert.Equal(t, {{ .packageName }}{{ .returnType }}(internal.GenerateTest{{ .returnType }}()), ms.{{ .fieldName }}())
	{{- else }}
	fillTest{{ .returnType }}(ms.{{ .fieldName }}())
	assert.Equal(t, generateTest{{ .returnType }}(), ms.{{ .fieldName }}())
	{{- end }}
}`

const accessorsPrimitiveTemplate = `// {{ .fieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .packageName }}{{ .returnType }} {
	return ms.{{ .origAccessor }}.{{ .originFieldName }}
}

// Set{{ .fieldName }} replaces the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) Set{{ .fieldName }}(v {{ .returnType }}) {
	ms.{{ .stateAccessor }}.AssertMutable()
	ms.{{ .origAccessor }}.{{ .originFieldName }} = v
}`

const setPrimitiveTestValueTemplate = `tv.orig.{{ .originFieldName }} = {{ .testValue }}`

const copyOrigPrimitiveTemplate = `dest.{{ .originFieldName }} = src.{{ .originFieldName }}`

const accessorsPrimitiveSliceTemplate = `// {{ .fieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .packageName }}{{ .returnType }} {
	return {{ .packageName }}{{ .returnType }}(internal.New{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .fieldName }}, ms.state))
}`

const oneOfTypeAccessorTemplate = `// {{ .typeFuncName }} returns the type of the {{ .lowerOriginFieldName }} for this {{ .structName }}.
// Calling this function on zero-initialized {{ .structName }} will cause a panic.
func (ms {{ .structName }}) {{ .typeFuncName }}() {{ .typeName }} {
	switch ms.{{ .origAccessor }}.{{ .originFieldName }}.(type) {
		{{- range .values }}
		{{ .GenerateTypeSwitchCase $.baseStruct $.oneOfField }}
		{{- end }}
	}
	return {{ .typeName }}Empty
}

{{ range .values }}
{{ .GenerateAccessors $.baseStruct $.oneOfField }}
{{ end }}`

const oneOfTypeAccessorTestTemplate = `func Test{{ .structName }}_{{ .typeFuncName }}(t *testing.T) {
	tv := New{{ .structName }}()
	assert.Equal(t, {{ .typeName }}Empty, tv.{{ .typeFuncName }}())
}

{{ range .values -}}
{{ .GenerateTests $.baseStruct $.oneOfField }}
{{ end }}
`

const oneOfTypeCopyOrigTestTemplate = `switch t := src.{{ .originFieldName }}.(type) {
{{- range .values }}
{{ .GenerateCopyOrig $.baseStruct $.oneOfField }}
{{- end }}
}`

const accessorsOneOfMessageTemplate = `// {{ .fieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
//
// Calling this function when {{ .originOneOfTypeFuncName }}() != {{ .typeName }} returns an invalid
// zero-initialized instance of {{ .returnType }}. Note that using such {{ .returnType }} instance can cause panic.
//
// Calling this function on zero-initialized {{ .structName }} will cause a panic.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .returnType }} {
	v, ok := ms.orig.Get{{ .originOneOfFieldName }}().(*{{ .originStructType }})
	if !ok {
		return {{ .returnType }}{}
	}
	return new{{ .returnType }}(v.{{ .fieldName }}, ms.state)
}

// SetEmpty{{ .fieldName }} sets an empty {{ .lowerFieldName }} to this {{ .structName }}.
//
// After this, {{ .originOneOfTypeFuncName }}() function will return {{ .typeName }}".
//
// Calling this function on zero-initialized {{ .structName }} will cause a panic.
func (ms {{ .structName }}) SetEmpty{{ .fieldName }}() {{ .returnType }} {
	ms.state.AssertMutable()
	val := &{{ .originFieldPackageName }}.{{ .fieldName }}{}
	ms.orig.{{ .originOneOfFieldName }} = &{{ .originStructType }}{{ "{" }}{{ .fieldName }}: val}
	return new{{ .returnType }}(val, ms.state)
}`

const accessorsOneOfMessageTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	fillTest{{ .returnType }}(ms.SetEmpty{{ .fieldName }}())
	assert.Equal(t, {{ .typeName }}, ms.{{ .originOneOfTypeFuncName }}())
	assert.Equal(t, generateTest{{ .returnType }}(), ms.{{ .fieldName }}())
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { new{{ .structName }}(&{{ .originStructName }}{}, &sharedState).SetEmpty{{ .fieldName }}() })
}

func Test{{ .structName }}_CopyTo_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	fillTest{{ .returnType }}(ms.SetEmpty{{ .fieldName }}())
	dest := New{{ .structName }}()
	ms.CopyTo(dest)
	assert.Equal(t, ms, dest)
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { ms.CopyTo(new{{ .structName }}(&{{ .originStructName }}{}, &sharedState)) })
}
`

const oneOfMessageSetTestFieldTemplate = `tv.orig.{{ .originOneOfFieldName }} = &{{ .originStructName }}_{{ .fieldName -}}{ 
{{- .fieldName }}: &{{ .originFieldPackageName }}.{{ .fieldName }}{}}
fillTest{{ .returnType }}(new{{ .returnType }}(tv.orig.Get{{ .returnType }}(), tv.state))`

const copyToValueOneOfMessageTemplate = `	case *{{ .originStructType }}:
		{{ .lowerFieldName }} := &{{ .originFieldPackageName}}.{{ .fieldName }}{}
		copyOrig{{ .returnType }}({{ .lowerFieldName }}, t.{{ .fieldName }})
		dest.{{ .originOneOfFieldName }} = &{{ .originStructType }}{
			{{ .fieldName }}: {{ .lowerFieldName }},
		}`

const accessorsOneOfPrimitiveTemplate = `// {{ .accessorFieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .accessorFieldName }}() {{ .returnType }} {
	return ms.orig.Get{{ .originFieldName }}()
}

// Set{{ .accessorFieldName }} replaces the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) Set{{ .accessorFieldName }}(v {{ .returnType }}) {
	ms.state.AssertMutable()
	ms.orig.{{ .originOneOfFieldName }} = &{{ .originStructType }}{
		{{ .originFieldName }}: v,
	}
}`

const accessorsOneOfPrimitiveTestTemplate = `func Test{{ .structName }}_{{ .accessorFieldName }}(t *testing.T) {
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
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { new{{ .structName }}(&{{ .originStructName }}{}, &sharedState).Set{{ .accessorFieldName }}({{ .testValue }}) })
}
`

const oneOfPrimitiveSetTestFieldTemplate = `tv.orig.{{ .originOneOfFieldName }} = &{{ .originStructName }}_{{ .originFieldName }}{
{{- .originFieldName }}: {{ .testValue }}}`

const oneOfPrimitiveCopyOrigFieldTemplate = `case *{{ .originStructName }}_{{ .originFieldName }}:
	dest.{{ .originOneOfFieldName }} = &{{ .originStructName }}_{{ .originFieldName }}{
{{- .originFieldName }}: t.{{ .originFieldName }}}`

const oneOfPrimitiveSwitchCaseTemplate = `case *{{ .originStructName }}_{{ .originFieldName }}:
	return {{ .typeName }}`

const accessorsPrimitiveTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	{{- if eq .returnType "bool" }}
	assert.{{- if eq .defaultVal "true" }}True{{- else }}False{{- end }}(t, ms.{{ .fieldName }}())
	{{- else if eq .returnType "float64" }}
	assert.InDelta(t, {{ .defaultVal }}, ms.{{ .fieldName }}(), 0.01)
	{{- else if and (eq .returnType "string") (eq .defaultVal "\"\"") }}
	assert.Empty(t, ms.{{ .fieldName }}())
	{{- else }}
	assert.Equal(t, {{ .defaultVal }}, ms.{{ .fieldName }}())
	{{- end }}
	ms.Set{{ .fieldName }}({{ .testValue }})
	{{- if eq .returnType "bool" }}
	assert.{{- if eq .testValue "true" }}True{{- else }}False{{- end }}(t, ms.{{ .fieldName }}())
	{{- else if eq .returnType "float64"}}
	assert.InDelta(t, {{ .testValue }}, ms.{{ .fieldName }}(), 0.01)
	{{- else if and (eq .returnType "string") (eq .testValue "\"\"") }}
	assert.Empty(t, ms.{{ .fieldName }}())
	{{- else }}
	assert.Equal(t, {{ .testValue }}, ms.{{ .fieldName }}())
	{{- end }}
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { new{{ .structName }}(&{{ .originStructName }}{}, &sharedState).Set{{ .fieldName }}({{ .testValue }}) })
}`

const accessorsPrimitiveTypedTemplate = `// {{ .fieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .packageName }}{{ .returnType }} {
	return {{ .packageName }}{{ .returnType }}(ms.orig.{{ .originFieldName }})
}

// Set{{ .fieldName }} replaces the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) Set{{ .fieldName }}(v {{ .packageName }}{{ .returnType }}) {
	ms.state.AssertMutable()
	ms.orig.{{ .originFieldName }} = {{ .rawType }}(v)
}`

const accessorsPrimitiveTypedTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	assert.Equal(t, {{ .packageName }}{{ .returnType }}({{ .defaultVal }}), ms.{{ .fieldName }}())
	testVal{{ .fieldName }} := {{ .packageName }}{{ .returnType }}({{ .testValue }})
	ms.Set{{ .fieldName }}(testVal{{ .fieldName }})
	assert.Equal(t, testVal{{ .fieldName }}, ms.{{ .fieldName }}())
}`

const accessorsPrimitiveSliceTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	assert.Equal(t, {{ .defaultVal }}, ms.{{ .fieldName }}().AsRaw())
	ms.{{ .fieldName }}().FromRaw({{ .testValue }})
	assert.Equal(t, {{ .testValue }}, ms.{{ .fieldName }}().AsRaw())
}`

const accessorsOptionalPrimitiveValueTemplate = `// {{ .fieldName }} returns the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .returnType }} {
	return ms.orig.Get{{ .fieldName }}()
}

// Has{{ .fieldName }} returns true if the {{ .structName }} contains a
// {{ .fieldName }} value, false otherwise.
func (ms {{ .structName }}) Has{{ .fieldName }}() bool {
	return ms.orig.{{ .fieldName }}_ != nil
}

// Set{{ .fieldName }} replaces the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) Set{{ .fieldName }}(v {{ .returnType }}) {
	ms.state.AssertMutable()
	ms.orig.{{ .fieldName }}_ = &{{ .originStructType }}{{ "{" }}{{ .fieldName }}: v}
}

// Remove{{ .fieldName }} removes the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) Remove{{ .fieldName }}() {
	ms.state.AssertMutable()
	ms.orig.{{ .fieldName }}_ = nil
}`

const accessorsOptionalPrimitiveTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
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

const optionalPrimitiveSetTestTemplate = `tv.orig.{{ .fieldName }}_ = &{{ .originStructType }}{
{{- .fieldName }}: {{ .testValue }}}`

const optionalPrimitiveCopyOrigTemplate = `if src{{ .fieldName }}, ok := src.{{ .fieldName }}_.(*{{ .originStructType }}); ok {
	dest{{ .fieldName }}, ok := dest.{{ .fieldName }}_.(*{{ .originStructType }})
	if !ok {
		dest{{ .fieldName }} = &{{ .originStructType }}{}
		dest.{{ .fieldName }}_ = dest{{ .fieldName }}
	}
	dest{{ .fieldName }}.{{ .fieldName }} = src{{ .fieldName }}.{{ .fieldName }}
} else {
	dest.{{ .fieldName }}_ = nil
}`

type baseField interface {
	GenerateAccessors(ms *messageValueStruct) string

	GenerateAccessorsTest(ms *messageValueStruct) string

	GenerateSetWithTestValue(ms *messageValueStruct) string

	GenerateCopyOrig(ms *messageValueStruct) string
}

type sliceField struct {
	fieldName       string
	originFieldName string
	returnSlice     baseSlice
	hideAccessors   bool
}

func (sf *sliceField) GenerateAccessors(ms *messageValueStruct) string {
	if sf.hideAccessors {
		return ""
	}
	t := template.Must(template.New("accessorSliceTemplate").Parse(accessorSliceTemplate))
	return executeTemplate(t, sf.templateFields(ms))
}

func (sf *sliceField) GenerateAccessorsTest(ms *messageValueStruct) string {
	if sf.hideAccessors {
		return ""
	}
	t := template.Must(template.New("accessorsSliceTestTemplate").Parse(accessorsSliceTestTemplate))
	return executeTemplate(t, sf.templateFields(ms))
}

func (sf *sliceField) GenerateSetWithTestValue(ms *messageValueStruct) string {
	t := template.Must(template.New("setTestValueTemplate").Parse(setTestValueTemplate))
	return executeTemplate(t, sf.templateFields(ms))
}

func (sf *sliceField) GenerateCopyOrig(ms *messageValueStruct) string {
	t := template.Must(template.New("copyOrigSliceTemplate").Parse(copyOrigSliceTemplate))
	return executeTemplate(t, sf.templateFields(ms))
}

func (sf *sliceField) templateFields(ms *messageValueStruct) map[string]any {
	return map[string]any{
		"structName": ms.getName(),
		"fieldName":  sf.fieldName,
		"packageName": func() string {
			if sf.returnSlice.getPackageName() != ms.packageName {
				return sf.returnSlice.getPackageName() + "."
			}
			return ""
		}(),
		"returnType":         sf.returnSlice.getName(),
		"origAccessor":       origAccessor(ms.packageName),
		"stateAccessor":      stateAccessor(ms.packageName),
		"isCommon":           usedByOtherDataTypes(sf.returnSlice.getPackageName()),
		"isBaseStructCommon": usedByOtherDataTypes(ms.packageName),
		"originFieldName":    sf.origFieldName(),
	}
}

func (sf *sliceField) origFieldName() string {
	if sf.originFieldName == "" {
		return sf.fieldName
	}
	return sf.originFieldName
}

var _ baseField = (*sliceField)(nil)

type messageValueField struct {
	fieldName     string
	returnMessage *messageValueStruct
}

func (mf *messageValueField) GenerateAccessors(ms *messageValueStruct) string {
	t := template.Must(template.New("accessorsMessageValueTemplate").Parse(accessorsMessageValueTemplate))
	return executeTemplate(t, mf.templateFields(ms))
}

func (mf *messageValueField) GenerateAccessorsTest(ms *messageValueStruct) string {
	t := template.Must(template.New("accessorsMessageValueTestTemplate").Parse(accessorsMessageValueTestTemplate))
	return executeTemplate(t, mf.templateFields(ms))
}

func (mf *messageValueField) GenerateSetWithTestValue(ms *messageValueStruct) string {
	t := template.Must(template.New("setTestValueTemplate").Parse(setTestValueTemplate))
	return executeTemplate(t, mf.templateFields(ms))
}

func (mf *messageValueField) GenerateCopyOrig(ms *messageValueStruct) string {
	t := template.Must(template.New("copyOrigMessageTemplate").Parse(copyOrigMessageTemplate))
	return executeTemplate(t, mf.templateFields(ms))
}

func (mf *messageValueField) templateFields(ms *messageValueStruct) map[string]any {
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

var _ baseField = (*messageValueField)(nil)

type primitiveField struct {
	fieldName       string
	originFieldName string
	returnType      string
	defaultVal      string
	testVal         string
}

func (pf *primitiveField) GenerateAccessors(ms *messageValueStruct) string {
	t := template.Must(template.New("accessorsPrimitiveTemplate").Parse(accessorsPrimitiveTemplate))
	return executeTemplate(t, pf.templateFields(ms))
}

func (pf *primitiveField) GenerateAccessorsTest(ms *messageValueStruct) string {
	t := template.Must(template.New("accessorsPrimitiveTestTemplate").Parse(accessorsPrimitiveTestTemplate))
	return executeTemplate(t, pf.templateFields(ms))
}

func (pf *primitiveField) GenerateSetWithTestValue(ms *messageValueStruct) string {
	t := template.Must(template.New("setPrimitiveTestValueTemplate").Parse(setPrimitiveTestValueTemplate))
	return executeTemplate(t, pf.templateFields(ms))
}

func (pf *primitiveField) GenerateCopyOrig(ms *messageValueStruct) string {
	t := template.Must(template.New("copyOrigPrimitiveTemplate").Parse(copyOrigPrimitiveTemplate))
	return executeTemplate(t, pf.templateFields(ms))
}

func (pf *primitiveField) templateFields(ms *messageValueStruct) map[string]any {
	return map[string]any{
		"structName":       ms.getName(),
		"packageName":      "",
		"defaultVal":       pf.defaultVal,
		"fieldName":        pf.fieldName,
		"lowerFieldName":   strings.ToLower(pf.fieldName),
		"testValue":        pf.testVal,
		"returnType":       pf.returnType,
		"origAccessor":     origAccessor(ms.packageName),
		"stateAccessor":    stateAccessor(ms.packageName),
		"originStructName": ms.originFullName,
		"originFieldName":  pf.origFieldName(),
	}
}

func (pf *primitiveField) origFieldName() string {
	if pf.originFieldName == "" {
		return pf.fieldName
	}
	return pf.originFieldName
}

var _ baseField = (*primitiveField)(nil)

type primitiveType struct {
	structName  string
	packageName string
	rawType     string
	defaultVal  string
	testVal     string
}

// Types that has defined a custom type (e.g. "type Timestamp uint64")
type primitiveTypedField struct {
	fieldName       string
	originFieldName string
	returnType      *primitiveType
}

func (ptf *primitiveTypedField) GenerateAccessors(ms *messageValueStruct) string {
	t := template.Must(template.New("accessorsPrimitiveTypedTemplate").Parse(accessorsPrimitiveTypedTemplate))
	return executeTemplate(t, ptf.templateFields(ms))
}

func (ptf *primitiveTypedField) GenerateAccessorsTest(ms *messageValueStruct) string {
	t := template.Must(template.New("accessorsPrimitiveTypedTestTemplate").Parse(accessorsPrimitiveTypedTestTemplate))
	return executeTemplate(t, ptf.templateFields(ms))
}

func (ptf *primitiveTypedField) GenerateSetWithTestValue(ms *messageValueStruct) string {
	t := template.Must(template.New("setPrimitiveTestValueTemplate").Parse(setPrimitiveTestValueTemplate))
	return executeTemplate(t, ptf.templateFields(ms))
}

func (ptf *primitiveTypedField) GenerateCopyOrig(ms *messageValueStruct) string {
	t := template.Must(template.New("copyOrigPrimitiveTemplate").Parse(copyOrigPrimitiveTemplate))
	return executeTemplate(t, ptf.templateFields(ms))
}

func (ptf *primitiveTypedField) templateFields(ms *messageValueStruct) map[string]any {
	return map[string]any{
		"structName": ms.getName(),
		"defaultVal": ptf.returnType.defaultVal,
		"packageName": func() string {
			if ptf.returnType.packageName != ms.packageName {
				return ptf.returnType.packageName + "."
			}
			return ""
		}(),
		"returnType":      ptf.returnType.structName,
		"fieldName":       ptf.fieldName,
		"lowerFieldName":  strings.ToLower(ptf.fieldName),
		"testValue":       ptf.returnType.testVal,
		"rawType":         ptf.returnType.rawType,
		"originFieldName": ptf.origFieldName(),
	}
}

func (ptf *primitiveTypedField) origFieldName() string {
	if ptf.originFieldName == "" {
		return ptf.fieldName
	}
	return ptf.originFieldName
}

var _ baseField = (*primitiveTypedField)(nil)

// primitiveSliceField is used to generate fields for slice of primitive types
type primitiveSliceField struct {
	fieldName         string
	returnPackageName string
	returnType        string
	defaultVal        string
	rawType           string
	testVal           string
}

func (psf *primitiveSliceField) GenerateAccessors(ms *messageValueStruct) string {
	t := template.Must(template.New("accessorsPrimitiveSliceTemplate").Parse(accessorsPrimitiveSliceTemplate))
	return executeTemplate(t, psf.templateFields(ms))
}

func (psf *primitiveSliceField) GenerateAccessorsTest(ms *messageValueStruct) string {
	t := template.Must(template.New("accessorsPrimitiveSliceTestTemplate").Parse(accessorsPrimitiveSliceTestTemplate))
	return executeTemplate(t, psf.templateFields(ms))
}

func (psf *primitiveSliceField) GenerateSetWithTestValue(ms *messageValueStruct) string {
	t := template.Must(template.New("setPrimitiveTestValueTemplate").Parse(setPrimitiveTestValueTemplate))
	return executeTemplate(t, psf.templateFields(ms))
}

func (psf *primitiveSliceField) GenerateCopyOrig(ms *messageValueStruct) string {
	t := template.Must(template.New("copyOrigSliceTemplate").Parse(copyOrigSliceTemplate))
	return executeTemplate(t, psf.templateFields(ms))
}

func (psf *primitiveSliceField) templateFields(ms *messageValueStruct) map[string]any {
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

var _ baseField = (*primitiveSliceField)(nil)

type oneOfField struct {
	originFieldName            string
	typeName                   string
	testValueIdx               int
	values                     []oneOfValue
	omitOriginFieldNameInNames bool
}

func (of *oneOfField) GenerateAccessors(ms *messageValueStruct) string {
	t := template.Must(template.New("oneOfTypeAccessorTemplate").Parse(oneOfTypeAccessorTemplate))
	return executeTemplate(t, of.templateFields(ms))
}

func (of *oneOfField) typeFuncName() string {
	const typeSuffix = "Type"
	if of.omitOriginFieldNameInNames {
		return typeSuffix
	}
	return of.originFieldName + typeSuffix
}

func (of *oneOfField) GenerateAccessorsTest(ms *messageValueStruct) string {
	t := template.Must(template.New("oneOfTypeAccessorTestTemplate").Parse(oneOfTypeAccessorTestTemplate))
	return executeTemplate(t, of.templateFields(ms))
}

func (of *oneOfField) GenerateSetWithTestValue(ms *messageValueStruct) string {
	return of.values[of.testValueIdx].GenerateSetWithTestValue(ms, of)
}

func (of *oneOfField) GenerateCopyOrig(ms *messageValueStruct) string {
	t := template.Must(template.New("oneOfTypeCopyOrigTestTemplate").Parse(oneOfTypeCopyOrigTestTemplate))
	return executeTemplate(t, of.templateFields(ms))
}

func (of *oneOfField) templateFields(ms *messageValueStruct) map[string]any {
	return map[string]any{
		"baseStruct":           ms,
		"oneOfField":           of,
		"structName":           ms.getName(),
		"typeFuncName":         of.typeFuncName(),
		"typeName":             of.typeName,
		"originFieldName":      of.originFieldName,
		"lowerOriginFieldName": strings.ToLower(of.originFieldName),
		"origAccessor":         origAccessor(ms.packageName),
		"stateAccessor":        stateAccessor(ms.packageName),
		"values":               of.values,
		"originTypePrefix":     ms.originFullName + "_",
	}
}

var _ baseField = (*oneOfField)(nil)

type oneOfValue interface {
	GenerateAccessors(ms *messageValueStruct, of *oneOfField) string
	GenerateTests(ms *messageValueStruct, of *oneOfField) string
	GenerateSetWithTestValue(ms *messageValueStruct, of *oneOfField) string
	GenerateCopyOrig(ms *messageValueStruct, of *oneOfField) string
	GenerateTypeSwitchCase(ms *messageValueStruct, of *oneOfField) string
}

type oneOfPrimitiveValue struct {
	fieldName       string
	defaultVal      string
	testVal         string
	returnType      string
	originFieldName string
}

func (opv *oneOfPrimitiveValue) GenerateAccessors(ms *messageValueStruct, of *oneOfField) string {
	t := template.Must(template.New("accessorsOneOfPrimitiveTemplate").Parse(accessorsOneOfPrimitiveTemplate))
	return executeTemplate(t, opv.templateFields(ms, of))
}

func (opv *oneOfPrimitiveValue) GenerateTests(ms *messageValueStruct, of *oneOfField) string {
	t := template.Must(template.New("accessorsOneOfPrimitiveTestTemplate").Parse(accessorsOneOfPrimitiveTestTemplate))
	return executeTemplate(t, opv.templateFields(ms, of))
}

func (opv *oneOfPrimitiveValue) accessorFieldName(of *oneOfField) string {
	if of.omitOriginFieldNameInNames {
		return opv.fieldName
	}
	return opv.fieldName + of.originFieldName
}

func (opv *oneOfPrimitiveValue) GenerateSetWithTestValue(ms *messageValueStruct, of *oneOfField) string {
	t := template.Must(template.New("oneOfPrimitiveSetTestFieldTemplate").Parse(oneOfPrimitiveSetTestFieldTemplate))
	return executeTemplate(t, opv.templateFields(ms, of))
}

func (opv *oneOfPrimitiveValue) GenerateCopyOrig(ms *messageValueStruct, of *oneOfField) string {
	t := template.Must(template.New("oneOfPrimitiveCopyOrigFieldTemplate").Parse(oneOfPrimitiveCopyOrigFieldTemplate))
	return executeTemplate(t, opv.templateFields(ms, of))
}

func (opv *oneOfPrimitiveValue) GenerateTypeSwitchCase(ms *messageValueStruct, of *oneOfField) string {
	t := template.Must(template.New("oneOfPrimitiveCopyOrigFieldTemplate").Parse(oneOfPrimitiveSwitchCaseTemplate))
	return executeTemplate(t, opv.templateFields(ms, of))
}

func (opv *oneOfPrimitiveValue) templateFields(ms *messageValueStruct, of *oneOfField) map[string]any {
	return map[string]any{
		"structName":              ms.getName(),
		"defaultVal":              opv.defaultVal,
		"packageName":             "",
		"accessorFieldName":       opv.accessorFieldName(of),
		"testValue":               opv.testVal,
		"originOneOfTypeFuncName": of.typeFuncName(),
		"typeName":                of.typeName + opv.fieldName,
		"lowerFieldName":          strings.ToLower(opv.fieldName),
		"returnType":              opv.returnType,
		"originFieldName":         opv.originFieldName,
		"originOneOfFieldName":    of.originFieldName,
		"originStructName":        ms.originFullName,
		"originStructType":        ms.originFullName + "_" + opv.originFieldName,
	}
}

var _ oneOfValue = (*oneOfPrimitiveValue)(nil)

type oneOfMessageValue struct {
	fieldName              string
	originFieldPackageName string
	returnMessage          *messageValueStruct
}

func (omv *oneOfMessageValue) GenerateAccessors(ms *messageValueStruct, of *oneOfField) string {
	t := template.Must(template.New("accessorsOneOfMessageTemplate").Parse(accessorsOneOfMessageTemplate))
	return executeTemplate(t, omv.templateFields(ms, of))
}

func (omv *oneOfMessageValue) GenerateTests(ms *messageValueStruct, of *oneOfField) string {
	t := template.Must(template.New("accessorsOneOfMessageTestTemplate").Parse(accessorsOneOfMessageTestTemplate))
	return executeTemplate(t, omv.templateFields(ms, of))
}

func (omv *oneOfMessageValue) GenerateSetWithTestValue(ms *messageValueStruct, of *oneOfField) string {
	t := template.Must(template.New("oneOfMessageSetTestFieldTemplate").Parse(oneOfMessageSetTestFieldTemplate))
	return executeTemplate(t, omv.templateFields(ms, of))
}

func (omv *oneOfMessageValue) GenerateCopyOrig(ms *messageValueStruct, of *oneOfField) string {
	t := template.Must(template.New("copyToValueOneOfMessageTemplate").Parse(copyToValueOneOfMessageTemplate))
	return executeTemplate(t, omv.templateFields(ms, of))
}

func (omv *oneOfMessageValue) GenerateTypeSwitchCase(ms *messageValueStruct, of *oneOfField) string {
	t := template.Must(template.New("oneOfPrimitiveCopyOrigFieldTemplate").Parse(oneOfPrimitiveSwitchCaseTemplate))
	return executeTemplate(t, omv.templateFields(ms, of))
}

func (omv *oneOfMessageValue) templateFields(ms *messageValueStruct, of *oneOfField) map[string]any {
	return map[string]any{
		"fieldName":               omv.fieldName,
		"originFieldName":         omv.fieldName,
		"originOneOfFieldName":    of.originFieldName,
		"typeName":                of.typeName + omv.fieldName,
		"structName":              ms.getName(),
		"returnType":              omv.returnMessage.structName,
		"originOneOfTypeFuncName": of.typeFuncName(),
		"lowerFieldName":          strings.ToLower(omv.fieldName),
		"originFieldPackageName":  omv.originFieldPackageName,
		"originStructName":        ms.originFullName,
		"originStructType":        ms.originFullName + "_" + omv.fieldName,
	}
}

var _ oneOfValue = (*oneOfMessageValue)(nil)

type optionalPrimitiveValue struct {
	fieldName  string
	defaultVal string
	testVal    string
	returnType string
}

func (opv *optionalPrimitiveValue) GenerateAccessors(ms *messageValueStruct) string {
	t := template.Must(template.New("accessorsOptionalPrimitiveValueTemplate").Parse(accessorsOptionalPrimitiveValueTemplate))
	return executeTemplate(t, opv.templateFields(ms))
}

func (opv *optionalPrimitiveValue) GenerateAccessorsTest(ms *messageValueStruct) string {
	t := template.Must(template.New("accessorsOptionalPrimitiveTestTemplate").Parse(accessorsOptionalPrimitiveTestTemplate))
	return executeTemplate(t, opv.templateFields(ms))
}

func (opv *optionalPrimitiveValue) GenerateSetWithTestValue(ms *messageValueStruct) string {
	t := template.Must(template.New("optionalPrimitiveSetTestTemplate").Parse(optionalPrimitiveSetTestTemplate))
	return executeTemplate(t, opv.templateFields(ms))
}

func (opv *optionalPrimitiveValue) GenerateCopyOrig(ms *messageValueStruct) string {
	t := template.Must(template.New("optionalPrimitiveCopyOrigTemplate").Parse(optionalPrimitiveCopyOrigTemplate))
	return executeTemplate(t, opv.templateFields(ms))
}

func (opv *optionalPrimitiveValue) templateFields(ms *messageValueStruct) map[string]any {
	return map[string]any{
		"structName":       ms.getName(),
		"packageName":      "",
		"defaultVal":       opv.defaultVal,
		"fieldName":        opv.fieldName,
		"lowerFieldName":   strings.ToLower(opv.fieldName),
		"testValue":        opv.testVal,
		"returnType":       opv.returnType,
		"originStructName": ms.originFullName,
		"originStructType": ms.originFullName + "_" + opv.fieldName,
	}
}

var _ baseField = (*optionalPrimitiveValue)(nil)

func origAccessor(packageName string) string {
	if usedByOtherDataTypes(packageName) {
		return "getOrig()"
	}
	return "orig"
}

func stateAccessor(packageName string) string {
	if usedByOtherDataTypes(packageName) {
		return "getState()"
	}
	return "state"
}
