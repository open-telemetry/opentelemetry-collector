// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

import (
	"bytes"
	"strings"
	"text/template"
)

const accessorSliceTemplate = `// {{ .fieldName }} returns the {{ .fieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) {{ .fieldName }}() {{ .packageName }}{{ .returnType }} {
	{{- if .isCommon }}
	return {{ .packageName }}{{ .returnType }}(internal.New{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .fieldName }}
	{{- if .isBaseStructCommon -}}
	, internal.Get{{ .structName }}State(internal.{{ .structName }}(ms))
	{{- else -}}
	, ms.state
	{{- end -}}
	))
	{{- else }}
	return new{{ .returnType }}(&ms.{{ .origAccessor }}.{{ .fieldName }}, ms.state)
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
	{{ .returnType }}(&tv.orig.{{ .fieldName }}, tv.state))`

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
	return ms.{{ .origAccessor }}.{{ .fieldName }}
}

// Set{{ .fieldName }} replaces the {{ .lowerFieldName }} associated with this {{ .structName }}.
func (ms {{ .structName }}) Set{{ .fieldName }}(v {{ .returnType }}) {
	ms.{{ .stateAccessor }}.AssertMutable()
	ms.{{ .origAccessor }}.{{ .fieldName }} = v
}`

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
{{ end }}`

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
}`

const copyToValueOneOfMessageTemplate = `	case {{ .typeName }}:
		ms.{{ .fieldName }}().CopyTo(dest.SetEmpty{{ .fieldName }}())`

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
	assert.Equal(t, {{ .defaultVal }}, ms.{{ .accessorFieldName }}())
	ms.Set{{ .accessorFieldName }}({{ .testValue }})
	assert.Equal(t, {{ .testValue }}, ms.{{ .accessorFieldName }}())
	assert.Equal(t, {{ .typeName }}, ms.{{ .originOneOfTypeFuncName }}())
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { new{{ .structName }}(&{{ .originStructName }}{}, &sharedState).Set{{ .accessorFieldName }}({{ .testValue }}) })
}`

const accessorsPrimitiveTestTemplate = `func Test{{ .structName }}_{{ .fieldName }}(t *testing.T) {
	ms := New{{ .structName }}()
	assert.Equal(t, {{ .defaultVal }}, ms.{{ .fieldName }}())
	ms.Set{{ .fieldName }}({{ .testValue }})
	assert.Equal(t, {{ .testValue }}, ms.{{ .fieldName }}())
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
	assert.Equal(t, {{ .defaultVal }}, ms.{{ .fieldName }}())
	ms.Set{{ .fieldName }}({{ .testValue }})
	assert.True(t, ms.Has{{ .fieldName }}())
	assert.Equal(t, {{ .testValue }}, ms.{{ .fieldName }}())
	ms.Remove{{ .fieldName }}()
	assert.False(t, ms.Has{{ .fieldName }}())
}`

type baseField interface {
	GenerateAccessors(ms *messageValueStruct) string

	GenerateAccessorsTest(ms *messageValueStruct) string

	GenerateSetWithTestValue(ms *messageValueStruct) string

	GenerateCopyToValue(ms *messageValueStruct) string
}

type sliceField struct {
	fieldName   string
	returnSlice baseSlice
}

func (sf *sliceField) GenerateAccessors(ms *messageValueStruct) string {
	sb := &strings.Builder{}
	t := template.Must(template.New("accessorSliceTemplate").Parse(accessorSliceTemplate))
	if err := t.Execute(sb, sf.templateFields(ms)); err != nil {
		panic(err)
	}
	return sb.String()
}

func (sf *sliceField) GenerateAccessorsTest(ms *messageValueStruct) string {
	sb := &strings.Builder{}
	t := template.Must(template.New("accessorsSliceTestTemplate").Parse(accessorsSliceTestTemplate))
	if err := t.Execute(sb, sf.templateFields(ms)); err != nil {
		panic(err)
	}
	return sb.String()
}

func (sf *sliceField) GenerateSetWithTestValue(ms *messageValueStruct) string {
	sb := &strings.Builder{}
	t := template.Must(template.New("setTestValueTemplate").Parse(setTestValueTemplate))
	if err := t.Execute(sb, sf.templateFields(ms)); err != nil {
		panic(err)
	}
	return sb.String()
}

func (sf *sliceField) GenerateCopyToValue(*messageValueStruct) string {
	return "\tms." + sf.fieldName + "().CopyTo(dest." + sf.fieldName + "())"
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
		"origAccessor":       origAccessor(ms),
		"stateAccessor":      stateAccessor(ms),
		"isCommon":           usedByOtherDataTypes(sf.returnSlice.getPackageName()),
		"isBaseStructCommon": usedByOtherDataTypes(ms.packageName),
	}
}

var _ baseField = (*sliceField)(nil)

type messageValueField struct {
	fieldName     string
	returnMessage *messageValueStruct
}

func (mf *messageValueField) GenerateAccessors(ms *messageValueStruct) string {
	sb := &strings.Builder{}
	t := template.Must(template.New("accessorsMessageValueTemplate").Parse(accessorsMessageValueTemplate))
	if err := t.Execute(sb, mf.templateFields(ms)); err != nil {
		panic(err)
	}
	return sb.String()
}

func (mf *messageValueField) GenerateAccessorsTest(ms *messageValueStruct) string {
	sb := &strings.Builder{}
	t := template.Must(template.New("accessorsMessageValueTestTemplate").Parse(accessorsMessageValueTestTemplate))
	if err := t.Execute(sb, mf.templateFields(ms)); err != nil {
		panic(err)
	}
	return sb.String()
}

func (mf *messageValueField) GenerateSetWithTestValue(ms *messageValueStruct) string {
	sb := &strings.Builder{}
	t := template.Must(template.New("setTestValueTemplate").Parse(setTestValueTemplate))
	if err := t.Execute(sb, mf.templateFields(ms)); err != nil {
		panic(err)
	}
	return sb.String()
}

func (mf *messageValueField) GenerateCopyToValue(*messageValueStruct) string {
	return "\tms." + mf.fieldName + "().CopyTo(dest." + mf.fieldName + "())"
}

func (mf *messageValueField) templateFields(ms *messageValueStruct) map[string]any {
	return map[string]any{
		"isCommon":       usedByOtherDataTypes(mf.returnMessage.packageName),
		"structName":     ms.getName(),
		"fieldName":      mf.fieldName,
		"lowerFieldName": strings.ToLower(mf.fieldName),
		"returnType":     mf.returnMessage.getName(),
		"packageName": func() string {
			if mf.returnMessage.packageName != ms.packageName {
				return mf.returnMessage.packageName + "."
			}
			return ""
		}(),
		"origAccessor":  origAccessor(ms),
		"stateAccessor": stateAccessor(ms),
	}
}

var _ baseField = (*messageValueField)(nil)

type primitiveField struct {
	fieldName  string
	returnType string
	defaultVal string
	testVal    string
}

func (pf *primitiveField) GenerateAccessors(ms *messageValueStruct) string {
	sb := &strings.Builder{}
	t := template.Must(template.New("accessorsPrimitiveTemplate").Parse(accessorsPrimitiveTemplate))
	if err := t.Execute(sb, pf.templateFields(ms)); err != nil {
		panic(err)
	}
	return sb.String()
}

func (pf *primitiveField) GenerateAccessorsTest(ms *messageValueStruct) string {
	sb := &strings.Builder{}
	t := template.Must(template.New("accessorsPrimitiveTestTemplate").Parse(accessorsPrimitiveTestTemplate))
	if err := t.Execute(sb, pf.templateFields(ms)); err != nil {
		panic(err)
	}
	return sb.String()
}

func (pf *primitiveField) GenerateSetWithTestValue(*messageValueStruct) string {
	return "\ttv.orig." + pf.fieldName + " = " + pf.testVal
}

func (pf *primitiveField) GenerateCopyToValue(*messageValueStruct) string {
	return "\tdest.Set" + pf.fieldName + "(ms." + pf.fieldName + "())"
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
		"origAccessor":     origAccessor(ms),
		"stateAccessor":    stateAccessor(ms),
		"originStructName": ms.originFullName,
	}
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
	sb := &strings.Builder{}
	t := template.Must(template.New("accessorsPrimitiveTypedTemplate").Parse(accessorsPrimitiveTypedTemplate))
	if err := t.Execute(sb, ptf.templateFields(ms)); err != nil {
		panic(err)
	}
	return sb.String()
}

func (ptf *primitiveTypedField) GenerateAccessorsTest(ms *messageValueStruct) string {
	sb := &strings.Builder{}
	t := template.Must(template.New("accessorsPrimitiveTypedTestTemplate").Parse(accessorsPrimitiveTypedTestTemplate))
	if err := t.Execute(sb, ptf.templateFields(ms)); err != nil {
		panic(err)
	}
	return sb.String()
}

func (ptf *primitiveTypedField) GenerateSetWithTestValue(*messageValueStruct) string {
	originFieldName := ptf.fieldName
	if ptf.originFieldName != "" {
		originFieldName = ptf.originFieldName
	}
	return "\ttv.orig." + originFieldName + " = " + ptf.returnType.testVal
}

func (ptf *primitiveTypedField) GenerateCopyToValue(*messageValueStruct) string {
	return "\tdest.Set" + ptf.fieldName + "(ms." + ptf.fieldName + "())"
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
		"returnType":     ptf.returnType.structName,
		"fieldName":      ptf.fieldName,
		"lowerFieldName": strings.ToLower(ptf.fieldName),
		"testValue":      ptf.returnType.testVal,
		"rawType":        ptf.returnType.rawType,
		"originFieldName": func() string {
			if ptf.originFieldName == "" {
				return ptf.fieldName
			}
			return ptf.originFieldName
		}(),
	}
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
	sb := &strings.Builder{}
	t := template.Must(template.New("accessorsPrimitiveSliceTemplate").Parse(accessorsPrimitiveSliceTemplate))
	if err := t.Execute(sb, psf.templateFields(ms)); err != nil {
		panic(err)
	}
	return sb.String()
}

func (psf *primitiveSliceField) GenerateAccessorsTest(ms *messageValueStruct) string {
	sb := &strings.Builder{}
	t := template.Must(template.New("accessorsPrimitiveSliceTestTemplate").Parse(accessorsPrimitiveSliceTestTemplate))
	if err := t.Execute(sb, psf.templateFields(ms)); err != nil {
		panic(err)
	}
	return sb.String()
}

func (psf *primitiveSliceField) GenerateSetWithTestValue(*messageValueStruct) string {
	return "\ttv.orig." + psf.fieldName + " = " + psf.testVal
}

func (psf *primitiveSliceField) GenerateCopyToValue(*messageValueStruct) string {
	return "\tms." + psf.fieldName + "().CopyTo(dest." + psf.fieldName + "())"
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
		"returnType":     psf.returnType,
		"defaultVal":     psf.defaultVal,
		"fieldName":      psf.fieldName,
		"lowerFieldName": strings.ToLower(psf.fieldName),
		"testValue":      psf.testVal,
		"origAccessor":   origAccessor(ms),
		"stateAccessor":  stateAccessor(ms),
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
	sb := &strings.Builder{}
	t := template.Must(template.New("oneOfTypeAccessorTemplate").Parse(oneOfTypeAccessorTemplate))
	if err := t.Execute(sb, of.templateFields(ms)); err != nil {
		panic(err)
	}
	return sb.String()
}

func (of *oneOfField) typeFuncName() string {
	const typeSuffix = "Type"
	if of.omitOriginFieldNameInNames {
		return typeSuffix
	}
	return of.originFieldName + typeSuffix
}

func (of *oneOfField) GenerateAccessorsTest(ms *messageValueStruct) string {
	sb := &strings.Builder{}
	t := template.Must(template.New("oneOfTypeAccessorTestTemplate").Parse(oneOfTypeAccessorTestTemplate))
	if err := t.Execute(sb, of.templateFields(ms)); err != nil {
		panic(err)
	}
	return sb.String()
}

func (of *oneOfField) GenerateSetWithTestValue(ms *messageValueStruct) string {
	return of.values[of.testValueIdx].GenerateSetWithTestValue(ms, of)
}

func (of *oneOfField) GenerateCopyToValue(ms *messageValueStruct) string {
	sb := &bytes.Buffer{}
	sb.WriteString("\tswitch ms." + of.typeFuncName() + "() {\n")
	for _, v := range of.values {
		v.GenerateCopyToValue(ms, of, sb)
	}
	sb.WriteString("\t}\n")
	return sb.String()
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
		"origAccessor":         origAccessor(ms),
		"stateAccessor":        stateAccessor(ms),
		"values":               of.values,
		"originTypePrefix":     ms.originFullName + "_",
	}
}

var _ baseField = (*oneOfField)(nil)

type oneOfValue interface {
	GenerateAccessors(ms *messageValueStruct, of *oneOfField) string
	GenerateTests(ms *messageValueStruct, of *oneOfField) string
	GenerateSetWithTestValue(ms *messageValueStruct, of *oneOfField) string
	GenerateCopyToValue(ms *messageValueStruct, of *oneOfField, sb *bytes.Buffer)
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
	sb := &strings.Builder{}
	t := template.Must(template.New("accessorsOneOfPrimitiveTemplate").Parse(accessorsOneOfPrimitiveTemplate))
	if err := t.Execute(sb, opv.templateFields(ms, of)); err != nil {
		panic(err)
	}
	return sb.String()
}

func (opv *oneOfPrimitiveValue) GenerateTests(ms *messageValueStruct, of *oneOfField) string {
	sb := &strings.Builder{}
	t := template.Must(template.New("accessorsOneOfPrimitiveTestTemplate").Parse(accessorsOneOfPrimitiveTestTemplate))
	if err := t.Execute(sb, opv.templateFields(ms, of)); err != nil {
		panic(err)
	}
	sb.WriteString("\n")
	return sb.String()
}

func (opv *oneOfPrimitiveValue) accessorFieldName(of *oneOfField) string {
	if of.omitOriginFieldNameInNames {
		return opv.fieldName
	}
	return opv.fieldName + of.originFieldName
}

func (opv *oneOfPrimitiveValue) GenerateSetWithTestValue(ms *messageValueStruct, of *oneOfField) string {
	return "\ttv.orig." + of.originFieldName + " = &" + ms.originFullName + "_" + opv.originFieldName + "{" + opv.
		originFieldName + ":" + opv.testVal + "}"
}

func (opv *oneOfPrimitiveValue) GenerateCopyToValue(_ *messageValueStruct, of *oneOfField, sb *bytes.Buffer) {
	sb.WriteString("\tcase " + of.typeName + opv.fieldName + ":\n")
	sb.WriteString("\tdest.Set" + opv.accessorFieldName(of) + "(ms." + opv.accessorFieldName(of) + "())\n")
}

func (opv *oneOfPrimitiveValue) GenerateTypeSwitchCase(ms *messageValueStruct, of *oneOfField) string {
	return "\tcase *" + ms.originFullName + "_" + opv.originFieldName + ":\n" +
		"\t\treturn " + of.typeName + opv.fieldName
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
	sb := &strings.Builder{}
	t := template.Must(template.New("accessorsOneOfMessageTemplate").Parse(accessorsOneOfMessageTemplate))
	if err := t.Execute(sb, omv.templateFields(ms, of)); err != nil {
		panic(err)
	}
	return sb.String()
}

func (omv *oneOfMessageValue) GenerateTests(ms *messageValueStruct, of *oneOfField) string {
	sb := &strings.Builder{}
	t := template.Must(template.New("accessorsOneOfMessageTestTemplate").Parse(accessorsOneOfMessageTestTemplate))
	if err := t.Execute(sb, omv.templateFields(ms, of)); err != nil {
		panic(err)
	}
	sb.WriteString("\n")
	return sb.String()
}

func (omv *oneOfMessageValue) GenerateSetWithTestValue(ms *messageValueStruct, of *oneOfField) string {
	return "\ttv.orig." + of.originFieldName + " = &" + ms.originFullName + "_" + omv.fieldName + "{" + omv.
		fieldName + ": &" + omv.originFieldPackageName + "." + omv.fieldName + "{}}\n" +
		"\tfillTest" + omv.returnMessage.structName + "(new" + omv.fieldName + "(tv.orig.Get" + omv.fieldName + "(), tv.state))"
}

func (omv *oneOfMessageValue) GenerateCopyToValue(ms *messageValueStruct, of *oneOfField, sb *bytes.Buffer) {
	t := template.Must(template.New("copyToValueOneOfMessageTemplate").Parse(copyToValueOneOfMessageTemplate))
	if err := t.Execute(sb, omv.templateFields(ms, of)); err != nil {
		panic(err)
	}
	sb.WriteString("\n")
}

func (omv *oneOfMessageValue) GenerateTypeSwitchCase(ms *messageValueStruct, of *oneOfField) string {
	return "\tcase *" + ms.originFullName + "_" + omv.fieldName + ":\n" +
		"\t\treturn " + of.typeName + omv.fieldName
}

func (omv *oneOfMessageValue) templateFields(ms *messageValueStruct, of *oneOfField) map[string]any {
	return map[string]any{
		"fieldName":               omv.fieldName,
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
	sb := &strings.Builder{}
	t := template.Must(template.New("accessorsOptionalPrimitiveValueTemplate").Parse(accessorsOptionalPrimitiveValueTemplate))
	if err := t.Execute(sb, opv.templateFields(ms)); err != nil {
		panic(err)
	}
	return sb.String()
}

func (opv *optionalPrimitiveValue) GenerateAccessorsTest(ms *messageValueStruct) string {
	sb := &strings.Builder{}
	t := template.Must(template.New("accessorsOptionalPrimitiveTestTemplate").Parse(accessorsOptionalPrimitiveTestTemplate))
	if err := t.Execute(sb, opv.templateFields(ms)); err != nil {
		panic(err)
	}
	sb.WriteString("\n")
	return sb.String()
}

func (opv *optionalPrimitiveValue) GenerateSetWithTestValue(ms *messageValueStruct) string {
	return "\ttv.orig." + opv.fieldName + "_ = &" + ms.originFullName + "_" + opv.fieldName + "{" + opv.fieldName + ":" + opv.testVal + "}"
}

func (opv *optionalPrimitiveValue) GenerateCopyToValue(*messageValueStruct) string {
	return "if ms.Has" + opv.fieldName + "(){\n" +
		"\tdest.Set" + opv.fieldName + "(ms." + opv.fieldName + "())\n" +
		"}\n"
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

func origAccessor(bs *messageValueStruct) string {
	if usedByOtherDataTypes(bs.packageName) {
		return "getOrig()"
	}
	return "orig"
}

func stateAccessor(bs *messageValueStruct) string {
	if usedByOtherDataTypes(bs.packageName) {
		return "getState()"
	}
	return "state"
}
