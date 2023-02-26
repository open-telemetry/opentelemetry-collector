// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

import (
	"bytes"
	"text/template"
)

const sliceTemplate = `// {{ .structName }} logically represents a slice of {{ .elementName }}.
//
// This is a reference type. If passed by value and callee modifies it, the
// caller will see the modification.
//
// Must use New{{ .structName }} function to create new instances.
// Important: zero-initialized instance is not valid for use.
type {{ .structName }} struct {
	orig *[]{{ .originElementType }}
}

func new{{ .structName }}(orig *[]{{ .originElementType }}) {{ .structName }} {
	return {{ .structName }}{orig}
}

// New{{ .structName }} creates a {{ .structName }} with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func New{{ .structName }}() {{ .structName }} {
	orig := []{{ .originElementType }}(nil)
	return new{{ .structName }}(&orig)
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "New{{ .structName }}()".
func (es {{ .structName }}) Len() int {
	return len(*es.orig)
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
	oldCap := cap(*es.orig)
	if newCap <= oldCap {
		return
	}

	newOrig := make([]{{ .originElementType }}, len(*es.orig), newCap)
	copy(newOrig, *es.orig)
	*es.orig = newOrig
}

// AppendEmpty will append to the end of the slice an empty {{ .elementName }}.
// It returns the newly added {{ .elementName }}.
func (es {{ .structName }}) AppendEmpty() {{ .elementName }} {
	*es.orig = append(*es.orig, {{ .emptyOriginElement }})
	return es.At(es.Len() - 1)
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es {{ .structName }}) MoveAndAppendTo(dest {{ .structName }}) {
	if *dest.orig == nil {
		// We can simply move the entire vector and avoid any allocations.
		*dest.orig = *es.orig
	} else {
		*dest.orig = append(*dest.orig, *es.orig...)
	}
	*es.orig = nil
}

// RemoveIf calls f sequentially for each element present in the slice.
// If f returns true, the element is removed from the slice.
func (es {{ .structName }}) RemoveIf(f func({{ .elementName }}) bool) {
	newLen := 0
	for i := 0; i < len(*es.orig); i++ {
		if f(es.At(i)) {
			continue
		}
		if newLen == i {
			// Nothing to move, element is at the right place.
			newLen++
			continue
		}
		(*es.orig)[newLen] = (*es.orig)[i]
		newLen++
	}
	// TODO: Prevent memory leak by erasing truncated values.
	*es.orig = (*es.orig)[:newLen]
}


// CopyTo copies all elements from the current slice overriding the destination.
func (es {{ .structName }}) CopyTo(dest {{ .structName }}) {
	srcLen := es.Len()
	destCap := cap(*dest.orig)
	if srcLen <= destCap {
		(*dest.orig) = (*dest.orig)[:srcLen:destCap]

	{{- if eq .type "sliceOfPtrs" }}
		for i := range *es.orig {
			new{{ .elementName }}((*es.orig)[i]).CopyTo(new{{ .elementName }}((*dest.orig)[i]))
		}
		return
	}
	origs := make([]{{ .originName }}, srcLen)
	wrappers := make([]*{{ .originName }}, srcLen)
	for i := range *es.orig {
		wrappers[i] = &origs[i]
		new{{ .elementName }}((*es.orig)[i]).CopyTo(new{{ .elementName }}(wrappers[i]))
	}
	*dest.orig = wrappers

	{{- else }}
	} else {
		(*dest.orig) = make([]{{ .originElementType }}, srcLen)
	}
	for i := range *es.orig {
		{{ .newElement }}.CopyTo(new{{ .elementName }}(&(*dest.orig)[i]))
	}
	{{- end }}
}

{{ if eq .type "sliceOfPtrs" -}}
// Sort sorts the {{ .elementName }} elements within {{ .structName }} given the
// provided less function so that two instances of {{ .structName }}
// can be compared.
func (es {{ .structName }}) Sort(less func(a, b {{ .elementName }}) bool) {
	sort.SliceStable(*es.orig, func(i, j int) bool { return less(es.At(i), es.At(j)) })
}
{{- end }}`

const sliceTestTemplate = `func Test{{ .structName }}(t *testing.T) {
	es := New{{ .structName }}()
	assert.Equal(t, 0, es.Len())
	es = new{{ .structName }}(&[]{{ .originElementType }}{})
	assert.Equal(t, 0, es.Len())

	emptyVal := New{{ .elementName }}()
	testVal := generateTest{{ .elementName }}()
	for i := 0; i < 7; i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, es.At(i))
		fillTest{{ .elementName }}(el)
		assert.Equal(t, testVal, es.At(i))
	}
	assert.Equal(t, 7, es.Len())
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
	assert.Equal(t, es.Len(), cap(*es.orig))
	assert.Equal(t, generateTest{{ .structName }}(), es)

	// Test ensure larger capacity
	const ensureLargeLen = 9
	es.EnsureCapacity(ensureLargeLen)
	assert.Less(t, generateTest{{ .structName }}().Len(), ensureLargeLen)
	assert.Equal(t, ensureLargeLen, cap(*es.orig))
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
		return uintptr(unsafe.Pointer(a.orig)) < uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.True(t, uintptr(unsafe.Pointer(es.At(i-1).orig)) < uintptr(unsafe.Pointer(es.At(i).orig)))
	}
	es.Sort(func(a, b {{ .elementName }}) bool {
		return uintptr(unsafe.Pointer(a.orig)) > uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.True(t, uintptr(unsafe.Pointer(es.At(i-1).orig)) > uintptr(unsafe.Pointer(es.At(i).orig)))
	}
}
{{- end }}`

const sliceGenerateTest = `func generateTest{{ .structName }}() {{ .structName }} {
	es := New{{ .structName }}()
	fillTest{{ .structName }}(es)
	return es
}

func fillTest{{ .structName }}(es {{ .structName }}) {
	*es.orig = make([]{{ .originElementType }}, 7)
	for i := 0; i < 7; i++ {
		(*es.orig)[i] = {{ .emptyOriginElement }}
		fillTest{{ .elementName }}({{ .newElement }})
	}
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
	t := template.Must(template.New("sliceGenerateTest").Parse(sliceGenerateTest))
	if err := t.Execute(sb, ss.templateFields()); err != nil {
		panic(err)
	}
}

func (ss *sliceOfPtrs) templateFields() map[string]any {
	return map[string]any{
		"type":               "sliceOfPtrs",
		"structName":         ss.structName,
		"elementName":        ss.element.structName,
		"originName":         ss.element.originFullName,
		"originElementType":  "*" + ss.element.originFullName,
		"emptyOriginElement": "&" + ss.element.originFullName + "{}",
		"newElement":         "new" + ss.element.structName + "((*es.orig)[i])",
	}
}

func (ss *sliceOfPtrs) generateInternal(_ *bytes.Buffer) {}

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
	t := template.Must(template.New("sliceGenerateTest").Parse(sliceGenerateTest))
	if err := t.Execute(sb, ss.templateFields()); err != nil {
		panic(err)
	}
}

func (ss *sliceOfValues) templateFields() map[string]any {
	return map[string]any{
		"type":               "sliceOfValues",
		"structName":         ss.structName,
		"elementName":        ss.element.structName,
		"originName":         ss.element.originFullName,
		"originElementType":  ss.element.originFullName,
		"emptyOriginElement": ss.element.originFullName + "{}",
		"newElement":         "new" + ss.element.structName + "(&(*es.orig)[i])",
	}
}

func (ss *sliceOfValues) generateInternal(_ *bytes.Buffer) {}

var _ baseStruct = (*sliceOfValues)(nil)
