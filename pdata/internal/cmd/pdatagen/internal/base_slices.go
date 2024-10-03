// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

import (
	"bytes"
	"strings"
	"text/template"
)

const sliceTemplate = `// {{ .structName }} logically represents a slice of {{ .elementName }}.
//
// This is a reference type. If passed by value and callee modifies it, the
// caller will see the modification.
//
// Must use New{{ .structName }} function to create new instances.
// Important: zero-initialized instance is not valid for use.
{{- if .isCommon }}
type {{ .structName }} internal.{{ .structName }}
{{- else }}
type {{ .structName }} struct {
	orig *[]{{ .originElementType }}
	state *internal.State
}
{{- end }}

func new{{ .structName }}(orig *[]{{ .originElementType }}, state *internal.State) {{ .structName }} {
	{{- if .isCommon }}
	return {{ .structName }}(internal.New{{ .structName }}(orig, state))
	{{- else }}
	return {{ .structName }}{orig: orig, state: state}
	{{- end }}
}

func (ms {{ .structName }}) getOrig() *[]{{ .originElementType }} {
	{{- if .isCommon }}
	return internal.GetOrig{{ .structName }}(internal.{{ .structName }}(ms))
	{{- else }}
	return ms.orig
	{{- end }}
}

func (ms {{ .structName }}) getState() *internal.State {
	{{- if .isCommon }}
	return internal.Get{{ .structName }}State(internal.{{ .structName }}(ms))
	{{- else }}
	return ms.state
	{{- end }}
}

// New{{ .structName }} creates a {{ .structName }} with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func New{{ .structName }}() {{ .structName }} {
	orig := []{{ .originElementType }}(nil)
	state := internal.StateMutable
	return new{{ .structName }}(&orig, &state)
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "New{{ .structName }}()".
func (es {{ .structName }}) Len() int {
	return len(*es.getOrig())
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
//   for i := 0; i < es.Len(); i++ {
//       e := es.At(i)
//       ... // Do something with the element
//   }
func (es {{ .structName }}) At(i int) {{ .elementName }} {
	return {{ .newElement }}
}

// EnsureCapacity is an operation that ensures the slice has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the slice capacity will be expanded to equal newCap.
//
// Here is how a new {{ .structName }} can be initialized:
//   es := New{{ .structName }}()
//   es.EnsureCapacity(4)
//   for i := 0; i < 4; i++ {
//       e := es.AppendEmpty()
//       // Here should set all the values for e.
//   }
func (es {{ .structName }}) EnsureCapacity(newCap int) {
	es.getState().AssertMutable()
	oldCap := cap(*es.getOrig())
	if newCap <= oldCap {
		return
	}

	newOrig := make([]{{ .originElementType }}, len(*es.getOrig()), newCap)
	copy(newOrig, *es.getOrig())
	*es.getOrig() = newOrig
}

// AppendEmpty will append to the end of the slice an empty {{ .elementName }}.
// It returns the newly added {{ .elementName }}.
func (es {{ .structName }}) AppendEmpty() {{ .elementName }} {
	es.getState().AssertMutable()
	*es.getOrig() = append(*es.getOrig(), {{ .emptyOriginElement }})
	return es.At(es.Len() - 1)
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es {{ .structName }}) MoveAndAppendTo(dest {{ .structName }}) {
	es.getState().AssertMutable()
	dest.getState().AssertMutable()
	if *dest.getOrig() == nil {
		// We can simply move the entire vector and avoid any allocations.
		*dest.getOrig() = *es.getOrig()
	} else {
		*dest.getOrig() = append(*dest.getOrig(), *es.getOrig()...)
	}
	*es.getOrig() = nil
}

// RemoveIf calls f sequentially for each element present in the slice.
// If f returns true, the element is removed from the slice.
func (es {{ .structName }}) RemoveIf(f func({{ .elementName }}) bool) {
	es.getState().AssertMutable()
	newLen := 0
	for i := 0; i < len(*es.getOrig()); i++ {
		if f(es.At(i)) {
			continue
		}
		if newLen == i {
			// Nothing to move, element is at the right place.
			newLen++
			continue
		}
		(*es.getOrig())[newLen] = (*es.getOrig())[i]
		newLen++
	}
	*es.getOrig() = (*es.getOrig())[:newLen]
}


// CopyTo copies all elements from the current slice overriding the destination.
func (es {{ .structName }}) CopyTo(dest {{ .structName }}) {
	dest.getState().AssertMutable()
	srcLen := es.Len()
	destCap := cap(*dest.getOrig())
	if srcLen <= destCap {
		(*dest.getOrig()) = (*dest.getOrig())[:srcLen:destCap]

	{{- if eq .type "sliceOfPtrs" }}
		for i := range *es.getOrig() {
			new{{ .elementName }}((*es.getOrig())[i], es.getState()).CopyTo(new{{ .elementName }}((*dest.getOrig())[i], dest.getState()))
		}
		return
	}
	origs := make([]{{ .originName }}, srcLen)
	wrappers := make([]*{{ .originName }}, srcLen)
	for i := range *es.getOrig() {
		wrappers[i] = &origs[i]
		new{{ .elementName }}((*es.getOrig())[i], es.getState()).CopyTo(new{{ .elementName }}(wrappers[i], dest.getState()))
	}
	*dest.getOrig() = wrappers

	{{- else }}
	} else {
		(*dest.getOrig()) = make([]{{ .originElementType }}, srcLen)
	}
	for i := range *es.getOrig() {
		{{ .newElement }}.CopyTo(new{{ .elementName }}(&(*dest.getOrig())[i], dest.getState()))
	}
	{{- end }}
}

{{ if eq .type "sliceOfPtrs" -}}
// Sort sorts the {{ .elementName }} elements within {{ .structName }} given the
// provided less function so that two instances of {{ .structName }}
// can be compared.
func (es {{ .structName }}) Sort(less func(a, b {{ .elementName }}) bool) {
	es.getState().AssertMutable()
	sort.SliceStable(*es.getOrig(), func(i, j int) bool { return less(es.At(i), es.At(j)) })
}
{{- end }}`

const sliceTestTemplate = `func Test{{ .structName }}(t *testing.T) {
	es := New{{ .structName }}()
	assert.Equal(t, 0, es.Len())
	state := internal.StateMutable
	es = new{{ .structName }}(&[]{{ .originElementType }}{}, &state)
	assert.Equal(t, 0, es.Len())

	emptyVal := New{{ .elementName }}()
	{{- if .isCommon }}
	testVal := {{ .elementName }}(internal.GenerateTest{{ .elementName }}())
	{{- else }}
	testVal := generateTest{{ .elementName }}()
	{{- end }}
	for i := 0; i < 7; i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, es.At(i))
		{{- if .isCommon }}
		internal.FillTest{{ .elementName }}(internal.{{ .elementName }}(el))
		{{- else }}
		fillTest{{ .elementName }}(el)
		{{- end }}
		assert.Equal(t, testVal, es.At(i))
	}
	assert.Equal(t, 7, es.Len())
}

func Test{{ .structName }}ReadOnly(t *testing.T) {
	sharedState := internal.StateReadOnly
	es := new{{ .structName }}(&[]{{ .originElementType }}{}, &sharedState)
	assert.Equal(t, 0, es.Len())
	assert.Panics(t, func() { es.AppendEmpty() })
	assert.Panics(t, func() { es.EnsureCapacity(2) })
	es2 := New{{ .structName }}()
	es.CopyTo(es2)
	assert.Panics(t, func() { es2.CopyTo(es) })
	assert.Panics(t, func() { es.MoveAndAppendTo(es2) })
	assert.Panics(t, func() { es2.MoveAndAppendTo(es) })
}

func Test{{ .structName }}_CopyTo(t *testing.T) {
	dest := New{{ .structName }}()
	// Test CopyTo to empty
	New{{ .structName }}().CopyTo(dest)
	assert.Equal(t, New{{ .structName }}(), dest)

	// Test CopyTo larger slice
	generateTest{{ .structName }}().CopyTo(dest)
	assert.Equal(t, generateTest{{ .structName }}(), dest)

	// Test CopyTo same size slice
	generateTest{{ .structName }}().CopyTo(dest)
	assert.Equal(t, generateTest{{ .structName }}(), dest)
}

func Test{{ .structName }}_EnsureCapacity(t *testing.T) {
	es := generateTest{{ .structName }}()

	// Test ensure smaller capacity.
	const ensureSmallLen = 4
	es.EnsureCapacity(ensureSmallLen)
	assert.Less(t, ensureSmallLen, es.Len())
	assert.Equal(t, es.Len(), cap(*es.getOrig()))
	assert.Equal(t, generateTest{{ .structName }}(), es)

	// Test ensure larger capacity
	const ensureLargeLen = 9
	es.EnsureCapacity(ensureLargeLen)
	assert.Less(t, generateTest{{ .structName }}().Len(), ensureLargeLen)
	assert.Equal(t, ensureLargeLen, cap(*es.getOrig()))
	assert.Equal(t, generateTest{{ .structName }}(), es)
}

func Test{{ .structName }}_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTest{{ .structName }}()
	dest := New{{ .structName }}()
	src := generateTest{{ .structName }}()
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTest{{ .structName }}(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTest{{ .structName }}(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTest{{ .structName }}().MoveAndAppendTo(dest)
	assert.Equal(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.Equal(t, expectedSlice.At(i), dest.At(i))
		assert.Equal(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func Test{{ .structName }}_RemoveIf(t *testing.T) {
	// Test RemoveIf on empty slice
	emptySlice := New{{ .structName }}()
	emptySlice.RemoveIf(func(el {{ .elementName }}) bool {
		t.Fail()
		return false
	})

	// Test RemoveIf
	filtered := generateTest{{ .structName }}()
	pos := 0
	filtered.RemoveIf(func(el {{ .elementName }}) bool {
		pos++
		return pos%3 == 0
	})
	assert.Equal(t, 5, filtered.Len())
}

{{ if eq .type "sliceOfPtrs" -}}
func Test{{ .structName }}_Sort(t *testing.T) {
	es := generateTest{{ .structName }}()
	es.Sort(func(a, b {{ .elementName }}) bool {
		return uintptr(unsafe.Pointer(a.getOrig())) < uintptr(unsafe.Pointer(b.getOrig()))
	})
	for i := 1; i < es.Len(); i++ {
		assert.Less(t, uintptr(unsafe.Pointer(es.At(i-1).getOrig())), uintptr(unsafe.Pointer(es.At(i).getOrig())))
	}
	es.Sort(func(a, b {{ .elementName }}) bool {
		return uintptr(unsafe.Pointer(a.getOrig())) > uintptr(unsafe.Pointer(b.getOrig()))
	})
	for i := 1; i < es.Len(); i++ {
		assert.Greater(t, uintptr(unsafe.Pointer(es.At(i-1).getOrig())), uintptr(unsafe.Pointer(es.At(i).getOrig())))
	}
}
{{- end }}

{{- if .isCommon }}
func generateTest{{ .structName }}() {{ .structName }} {
	return {{ .structName }}(internal.GenerateTest{{ .structName }}())
}
{{- end }}`

const sliceGenerateTest = `func {{ upperIfInternal "g" }}enerateTest{{ .structName }}() {{ .structName }} {
	{{- if .isCommon }}
	orig := []{{ .originElementType }}(nil)
	state := StateMutable
	{{- end }}
	es := New{{ .structName }}({{ if .isCommon }}&orig, &state{{ end }})
	{{ upperIfInternal "f" }}illTest{{ .structName }}(es)
	return es
}

func {{ upperIfInternal "f" }}illTest{{ .structName }}(es {{ .structName }}) {
	*es.orig = make([]{{ .originElementType }}, 7)
	for i := 0; i < 7; i++ {
		(*es.orig)[i] = {{ .emptyOriginElement }}
		{{ upperIfInternal "f" }}illTest{{ .elementName }}({{ upperIfInternal "n" }}ew{{ .elementName }}({{ .elementGetter }}, es.state))
	}
}`

const sliceValueAliasTemplate = `
type {{ .structName }} struct {
	orig *[]{{ .originElementType }}
	state *State
}

func GetOrig{{ .structName }}(ms {{ .structName }}) *[]{{ .originElementType }} {
	return ms.orig
}

func Get{{ .structName }}State(ms {{ .structName }}) *State {
	return ms.state
}

func New{{ .structName }}(orig *[]{{ .originElementType }}, state *State) {{ .structName }} {
	return {{ .structName }}{orig: orig, state: state}
}`

type baseSlice interface {
	getName() string
	getPackageName() string
}

// sliceOfPtrs generates code for a slice of pointer fields. The generated structs cannot be used from other packages.
type sliceOfPtrs struct {
	structName  string
	packageName string
	element     *messageValueStruct
}

func (ss *sliceOfPtrs) getName() string {
	return ss.structName
}

func (ss *sliceOfPtrs) getPackageName() string {
	return ss.packageName
}

func (ss *sliceOfPtrs) generateStruct(sb *bytes.Buffer) {
	t := template.Must(template.New("sliceTemplate").Parse(sliceTemplate))
	if err := t.Execute(sb, ss.templateFields()); err != nil {
		panic(err)
	}
}

func (ss *sliceOfPtrs) generateTests(sb *bytes.Buffer) {
	t := template.Must(template.New("sliceTestTemplate").Parse(sliceTestTemplate))
	if err := t.Execute(sb, ss.templateFields()); err != nil {
		panic(err)
	}
}

func (ss *sliceOfPtrs) generateTestValueHelpers(sb *bytes.Buffer) {
	funcs := template.FuncMap{
		"upperIfInternal": func(in string) string {
			if usedByOtherDataTypes(ss.packageName) {
				return strings.ToUpper(in)
			}
			return in
		},
	}
	t := template.Must(template.New("sliceGenerateTest").Funcs(funcs).Parse(sliceGenerateTest))
	if err := t.Execute(sb, ss.templateFields()); err != nil {
		panic(err)
	}
}

func (ss *sliceOfPtrs) templateFields() map[string]any {
	newElement := "new" + ss.element.structName + "((*es.getOrig())[i], es.getState())"
	if usedByOtherDataTypes(ss.packageName) {
		newElement = ss.element.structName + "(internal.New" + ss.element.structName + "((*es.getOrig())[i], es.getState()))"
	}
	return map[string]any{
		"type":               "sliceOfPtrs",
		"isCommon":           usedByOtherDataTypes(ss.packageName),
		"structName":         ss.structName,
		"elementName":        ss.element.structName,
		"originName":         ss.element.originFullName,
		"originElementType":  "*" + ss.element.originFullName,
		"emptyOriginElement": "&" + ss.element.originFullName + "{}",
		"newElement":         newElement,
		"elementGetter":      "(*es.orig)[i]",
	}
}

func (ss *sliceOfPtrs) generateInternal(sb *bytes.Buffer) {
	t := template.Must(template.New("sliceValueAliasTemplate").Parse(sliceValueAliasTemplate))
	if err := t.Execute(sb, ss.templateFields()); err != nil {
		panic(err)
	}
}

var _ baseStruct = (*sliceOfPtrs)(nil)

// sliceOfValues generates code for a slice of pointer fields. The generated structs cannot be used from other packages.
type sliceOfValues struct {
	structName  string
	packageName string
	element     *messageValueStruct
}

func (ss *sliceOfValues) getName() string {
	return ss.structName
}

func (ss *sliceOfValues) getPackageName() string {
	return ss.packageName
}

func (ss *sliceOfValues) generateStruct(sb *bytes.Buffer) {
	t := template.Must(template.New("sliceTemplate").Parse(sliceTemplate))
	if err := t.Execute(sb, ss.templateFields()); err != nil {
		panic(err)
	}
}

func (ss *sliceOfValues) generateTests(sb *bytes.Buffer) {
	t := template.Must(template.New("sliceTestTemplate").Parse(sliceTestTemplate))
	if err := t.Execute(sb, ss.templateFields()); err != nil {
		panic(err)
	}
}

func (ss *sliceOfValues) generateTestValueHelpers(sb *bytes.Buffer) {
	funcs := template.FuncMap{
		"upperIfInternal": func(in string) string {
			if usedByOtherDataTypes(ss.packageName) {
				return strings.ToUpper(in)
			}
			return in
		},
	}
	t := template.Must(template.New("sliceGenerateTest").Funcs(funcs).Parse(sliceGenerateTest))
	if err := t.Execute(sb, ss.templateFields()); err != nil {
		panic(err)
	}
}

func (ss *sliceOfValues) templateFields() map[string]any {
	newElement := "new" + ss.element.structName + "(&(*es.getOrig())[i], es.getState())"
	if usedByOtherDataTypes(ss.packageName) {
		newElement = ss.element.structName + "(internal.New" + ss.element.structName + "(&(*es.getOrig())[i], es.getState()))"
	}
	return map[string]any{
		"type":               "sliceOfValues",
		"structName":         ss.structName,
		"elementName":        ss.element.structName,
		"originName":         ss.element.originFullName,
		"originElementType":  ss.element.originFullName,
		"emptyOriginElement": ss.element.originFullName + "{}",
		"newElement":         newElement,
		"elementGetter":      "&(*es.orig)[i]",
	}
}

func (ss *sliceOfValues) generateInternal(*bytes.Buffer) {}

var _ baseStruct = (*sliceOfValues)(nil)
