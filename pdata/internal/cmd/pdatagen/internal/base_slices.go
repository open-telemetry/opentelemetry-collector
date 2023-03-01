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
	parent {{ .parent }}
}

{{- if .parentUnspecified }}
type stub{{ .structName }}Parent struct {
	orig *{{ .origType }}
}

func (s stub{{ .structName }}Parent) GetChildOrig() *{{ .origType }} {
	return s.orig
}

func (s stub{{ .structName }}Parent) EnsureMutability() {}

func (s stub{{ .structName }}Parent) GetState() *internal.State {
	state := internal.StateExclusive
	return &state
}
{{- end }}

func new{{ .structName }}FromOrig(orig *{{ .origType }}) {{ .structName }} {
	{{- if .parentUnspecified }}
	return {{ .structName }}{parent: stub{{ .structName }}Parent{orig: orig}}
	{{- else }}
	return {{ .structName }}{parent: new{{ .parent }}From{{ .parentField }}Orig(orig)}
	{{- end }}
}

func new{{ .structName }}FromParent(parent {{ .parent }}) {{ .structName }} {
	return {{ .structName }}{parent: parent}
}

func (es {{ .structName }}) getOrig() *{{ .origType }} {
	{{- if .parentUnspecified }}
	return es.parent.GetChildOrig()
	{{- else }}
	return es.parent.get{{ .parentField }}Orig()
	{{- end }}
}

func (es {{ .structName }}) ensureMutability() {
	es.parent.{{- if .parentUnspecified }}E{{ else }}e{{ end }}nsureMutability()
}

func (es {{ .structName }}) getState() *internal.State {
	return es.parent.{{ if .parentUnspecified }}G{{ else }}g{{ end }}etState()
}

func (es {{ .structName }}) refreshElementOrigState(i int) (*{{ .originName }}, *internal.State) {
	return {{ if eq .type "sliceOfValues" }}&{{ end }}(*es.getOrig())[i], es.getState()
}

// New{{ .structName }} creates a {{ .structName }} with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func New{{ .structName }}() {{ .structName }} {
	orig := {{ .origType }}(nil)
	return new{{ .structName }}FromOrig(&orig)
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
	es.ensureMutability()
	oldCap := cap(*es.getOrig())
	if newCap <= oldCap {
		return
	}

	newOrig := make({{ .origType }}, len(*es.getOrig()), newCap)
	copy(newOrig, *es.getOrig())
	*es.getOrig() = newOrig
}

// AppendEmpty will append to the end of the slice an empty {{ .elementName }}.
// It returns the newly added {{ .elementName }}.
func (es {{ .structName }}) AppendEmpty() {{ .elementName }} {
	es.ensureMutability()
	*es.getOrig() = append(*es.getOrig(), {{ .emptyOriginElement }})
	return es.At(es.Len() - 1)
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es {{ .structName }}) MoveAndAppendTo(dest {{ .structName }}) {
	es.ensureMutability()
	dest.ensureMutability()
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
	es.ensureMutability()
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
	// TODO: Prevent memory leak by erasing truncated values.
	*es.getOrig() = (*es.getOrig())[:newLen]
}


// CopyTo copies all elements from the current slice overriding the destination.
func (es {{ .structName }}) CopyTo(dest {{ .structName }}) {
	dest.ensureMutability()
	srcLen := es.Len()
	destCap := cap(*dest.getOrig())
	exclState := internal.StateExclusive 
	if srcLen <= destCap {
		(*dest.getOrig()) = (*dest.getOrig())[:srcLen:destCap]

	{{- if eq .type "sliceOfPtrs" }}
		for i := range *es.getOrig() {
			src{{ .elementName }} := {{ .elementName }}{&p{{ .elementName }}{orig: (*es.getOrig())[i], state: &exclState}} 
			dest{{ .elementName }} := {{ .elementName }}{&p{{ .elementName }}{orig: (*dest.getOrig())[i], 
state: &exclState}}
			src{{ .elementName }}.CopyTo(dest{{ .elementName }})
		}
		return
	}
	origs := make([]{{ .originName }}, srcLen)
	wrappers := make([]*{{ .originName }}, srcLen)
	for i := range *es.getOrig() {
		wrappers[i] = &origs[i]
		src{{ .elementName }} := {{ .elementName }}{&p{{ .elementName }}{orig: (*es.getOrig())[i], state: &exclState}} 
		dest{{ .elementName }} := {{ .elementName }}{&p{{ .elementName }}{orig: wrappers[i], state: &exclState}}
		src{{ .elementName }}.CopyTo(dest{{ .elementName }})
	}
	*dest.getOrig() = wrappers

	{{- else }}
	} else {
		(*dest.getOrig()) = make({{ .origType }}, srcLen)
	}
	for i := range *es.getOrig() {
		src{{ .elementName }} := {{ .elementName }}{&p{{ .elementName }}{orig: &(*es.getOrig())[i], state: &exclState}}
		dest{{ .elementName }} := {{ .elementName }}{&p{{ .elementName }}{orig: &(*dest.getOrig())[i], 
state: &exclState}}
		src{{ .elementName }}.CopyTo(dest{{ .elementName }})
	}
	{{- end }}
}

{{ if eq .type "sliceOfPtrs" -}}
// Sort sorts the {{ .elementName }} elements within {{ .structName }} given the
// provided less function so that two instances of {{ .structName }}
// can be compared.
func (es {{ .structName }}) Sort(less func(a, b {{ .elementName }}) bool) {
	es.ensureMutability()
	sort.SliceStable(*es.getOrig(), func(i, j int) bool { return less(es.At(i), es.At(j)) })
}
{{- end }}`

const sliceTestTemplate = `func Test{{ .structName }}(t *testing.T) {
	es := New{{ .structName }}()
	assert.Equal(t, 0, es.Len())

	emptyVal := New{{ .elementName }}()
	testVal := generateTest{{ .elementName }}()
	for i := 0; i < 7; i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal.getOrig(), es.At(i).getOrig())
		fillTest{{ .elementName }}(el)
		assert.Equal(t, testVal.getOrig(), es.At(i).getOrig())
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
		assert.Equal(t, expectedSlice.At(i).getOrig(), dest.At(i).getOrig())
		assert.Equal(t, expectedSlice.At(i).getOrig(), dest.At(i+expectedSlice.Len()).getOrig())
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
		assert.True(t, uintptr(unsafe.Pointer(es.At(i-1).getOrig())) < uintptr(unsafe.Pointer(es.At(i).getOrig())))
	}
	es.Sort(func(a, b {{ .elementName }}) bool {
		return uintptr(unsafe.Pointer(a.getOrig())) > uintptr(unsafe.Pointer(b.getOrig()))
	})
	for i := 1; i < es.Len(); i++ {
		assert.True(t, uintptr(unsafe.Pointer(es.At(i-1).getOrig())) > uintptr(unsafe.Pointer(es.At(i).getOrig())))
	}
}
{{- end }}`

const sliceGenerateTest = `func generateTest{{ .structName }}() {{ .structName }} {
	es := New{{ .structName }}()
	fillTest{{ .structName }}(es)
	return es
}

func fillTest{{ .structName }}(es {{ .structName }}) {
	*es.getOrig() = make({{ .origType }}, 7)
	for i := 0; i < 7; i++ {
		(*es.getOrig())[i] = {{ .emptyOriginElement }}
		fillTest{{ .elementName }}({{ .newElement }})
	}
}`

type baseSlice interface {
	getName() string
	getPackageName() string
	getParent() string
	getOrigType() string
}

// sliceOfPtrs generates code for a slice of pointer fields. The generated structs cannot be used from other packages.
type sliceOfPtrs struct {
	structName  string
	packageName string
	parent      string
	parentField string
	element     *messageValueStruct
}

func (ss *sliceOfPtrs) getName() string {
	return ss.structName
}

func (ss *sliceOfPtrs) getPackageName() string {
	return ss.packageName
}

func (ss *sliceOfPtrs) getParent() string {
	return ss.parent
}

func (ss *sliceOfPtrs) getOrigType() string {
	if ss.element == nil {
		return ""
	}
	return "[]*" + ss.element.originFullName
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
		"origType":           ss.getOrigType(),
		"emptyOriginElement": "&" + ss.element.originFullName + "{}",
		"newElement":         "new" + ss.element.structName + "((*es.getOrig())[i], es, i)",
		"parentUnspecified":  ss.parent == "",
		"parent": func() string {
			if ss.parent == "" {
				return "internal.SliceParent[*[]*" + ss.element.originFullName + "]"
			}
			return ss.parent
		}(),
		"parentField": ss.parentField,
	}
}

func (ss *sliceOfPtrs) generateInternal(_ *bytes.Buffer) {}

var _ baseStruct = (*sliceOfPtrs)(nil)

// sliceOfValues generates code for a slice of pointer fields. The generated structs cannot be used from other packages.
type sliceOfValues struct {
	structName  string
	packageName string
	parent      string
	element     *messageValueStruct
}

func (ss *sliceOfValues) getName() string {
	return ss.structName
}

func (ss *sliceOfValues) getPackageName() string {
	return ss.packageName
}

func (ss *sliceOfValues) getParent() string {
	return ss.parent
}

func (ss *sliceOfValues) getOrigType() string {
	return "[]" + ss.element.originFullName
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
		"origType":           ss.getOrigType(),
		"emptyOriginElement": ss.element.originFullName + "{}",
		"newElement":         "new" + ss.element.structName + "(&(*es.getOrig())[i], es, i)",
		"parentUnspecified":  ss.parent == "",
		"parent": func() string {
			if ss.parent == "" {
				return "internal.SliceParent[*[]" + ss.element.originFullName + "]"
			}
			return ss.parent
		}(),
	}
}

func (ss *sliceOfValues) generateInternal(_ *bytes.Buffer) {}

var _ baseStruct = (*sliceOfValues)(nil)
