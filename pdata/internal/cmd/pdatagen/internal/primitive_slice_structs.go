// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

import (
	"bytes"
	"strings"
	"text/template"
)

const primitiveSliceTemplate = `// {{ .structName }} represents a []{{ .itemType }} slice.
// The instance of {{ .structName }} can be assigned to multiple objects since it's immutable.
//
// Must use New{{ .structName }} function to create new instances.
// Important: zero-initialized instance is not valid for use.
type {{ .structName }} internal.{{ .structName }}

func (ms {{ .structName }}) getOrig() *[]{{ .itemType }} {
	return internal.GetOrig{{ .structName }}(internal.{{ .structName }}(ms))
}

func (ms {{ .structName }}) getState() *internal.State {
	return internal.Get{{ .structName }}State(internal.{{ .structName }}(ms))
}

// New{{ .structName }} creates a new empty {{ .structName }}.
func New{{ .structName }}() {{ .structName }} {
	orig := []{{ .itemType }}(nil)
	state := internal.StateMutable
	return {{ .structName }}(internal.New{{ .structName }}(&orig, &state))
}

// AsRaw returns a copy of the []{{ .itemType }} slice.
func (ms {{ .structName }}) AsRaw() []{{ .itemType }} {
	return copy{{ .structName }}(nil, *ms.getOrig())
}

// FromRaw copies raw []{{ .itemType }} into the slice {{ .structName }}.
func (ms {{ .structName }}) FromRaw(val []{{ .itemType }}) {
	ms.getState().AssertMutable()
	*ms.getOrig() = copy{{ .structName }}(*ms.getOrig(), val)
}

// Len returns length of the []{{ .itemType }} slice value.
// Equivalent of len({{ .lowerStructName }}).
func (ms {{ .structName }}) Len() int {
	return len(*ms.getOrig())
}

// At returns an item from particular index.
// Equivalent of {{ .lowerStructName }}[i].
func (ms {{ .structName }}) At(i int) {{ .itemType }} {
	return (*ms.getOrig())[i]
}

// SetAt sets {{ .itemType }} item at particular index.
// Equivalent of {{ .lowerStructName }}[i] = val
func (ms {{ .structName }}) SetAt(i int, val {{ .itemType }}) {
	ms.getState().AssertMutable()
	(*ms.getOrig())[i] = val
}

// EnsureCapacity ensures {{ .structName }} has at least the specified capacity.
// 1. If the newCap <= cap, then is no change in capacity.
// 2. If the newCap > cap, then the slice capacity will be expanded to the provided value which will be equivalent of:
//	buf := make([]{{ .itemType }}, len({{ .lowerStructName }}), newCap)
//	copy(buf, {{ .lowerStructName }})
//	{{ .lowerStructName }} = buf
func (ms {{ .structName }}) EnsureCapacity(newCap int) {
	ms.getState().AssertMutable()
	oldCap := cap(*ms.getOrig())
	if newCap <= oldCap {
		return
	}

	newOrig := make([]{{ .itemType }}, len(*ms.getOrig()), newCap)
	copy(newOrig, *ms.getOrig())
	*ms.getOrig() = newOrig
}

// Append appends extra elements to {{ .structName }}.
// Equivalent of {{ .lowerStructName }} = append({{ .lowerStructName }}, elms...)
func (ms {{ .structName }}) Append(elms ...{{ .itemType }}) {
	ms.getState().AssertMutable()
	*ms.getOrig() = append(*ms.getOrig(), elms...)
}

// MoveTo moves all elements from the current slice overriding the destination and
// resetting the current instance to its zero value.
func (ms {{ .structName }}) MoveTo(dest {{ .structName }}) {
	ms.getState().AssertMutable()
	dest.getState().AssertMutable()
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = nil
}

// CopyTo copies all elements from the current slice overriding the destination.
func (ms {{ .structName }}) CopyTo(dest {{ .structName }}) {
	dest.getState().AssertMutable()
	*dest.getOrig() = copy{{ .structName }}(*dest.getOrig(), *ms.getOrig())
}

func copy{{ .structName }}(dst, src []{{ .itemType }}) []{{ .itemType }} {
	dst = dst[:0]
	return append(dst, src...)
}`

const immutableSliceTestTemplate = `func TestNew{{ .structName }}(t *testing.T) {
	ms := New{{ .structName }}()
	assert.Equal(t, 0, ms.Len())
	ms.FromRaw([]{{ .itemType }}{ {{ .testOrigVal }} })
	assert.Equal(t, 3, ms.Len())
	assert.Equal(t, []{{ .itemType }}{ {{ .testOrigVal }} }, ms.AsRaw())
	ms.SetAt(1, {{ .itemType }}( {{ .testSetVal }} ))
	assert.Equal(t, []{{ .itemType }}{ {{ .testNewVal }} }, ms.AsRaw())
	ms.FromRaw([]{{ .itemType }}{ {{ index .testInterfaceOrigVal 2 }} })
	assert.Equal(t, 1, ms.Len())
	assert.Equal(t, {{ .itemType }}( {{ index .testInterfaceOrigVal 2 }} ), ms.At(0))

	cp := New{{ .structName }}()
	ms.CopyTo(cp)
	ms.SetAt(0, {{ .itemType }}( {{ index .testInterfaceOrigVal 1 }} ))
	assert.Equal(t, {{ .itemType }}( {{ index .testInterfaceOrigVal 1 }} ), ms.At(0))
	assert.Equal(t, {{ .itemType }}({{ index .testInterfaceOrigVal 2 }}), cp.At(0))
	ms.CopyTo(cp)
	assert.Equal(t, {{ .itemType }}({{ index .testInterfaceOrigVal 1 }}), cp.At(0))

	mv := New{{ .structName }}()
	ms.MoveTo(mv)
	assert.Equal(t, 0, ms.Len())
	assert.Equal(t, 1, mv.Len())
	assert.Equal(t, {{ .itemType }}({{ index .testInterfaceOrigVal 1 }}), mv.At(0))
	ms.FromRaw([]{{ .itemType }}{ {{ .testOrigVal }} })
	ms.MoveTo(mv)
	assert.Equal(t, 3, mv.Len())
	assert.Equal(t, {{ .itemType }}({{ index .testInterfaceOrigVal 0 }}), mv.At(0))
}

func Test{{ .structName }}ReadOnly(t *testing.T) {
	raw := []{{ .itemType }}{ {{ .testOrigVal }}}
	state := internal.StateReadOnly
	ms := {{ .structName }}(internal.New{{ .structName }}(&raw, &state))

	assert.Equal(t, 3, ms.Len())
	assert.Equal(t, {{ .itemType }}({{ index .testInterfaceOrigVal 0 }}), ms.At(0))
	assert.Panics(t, func() { ms.Append({{ index .testInterfaceOrigVal 0 }}) })
	assert.Panics(t, func() { ms.EnsureCapacity(2) })
	assert.Equal(t, raw, ms.AsRaw())
	assert.Panics(t, func() { ms.FromRaw(raw) })

	ms2 := New{{ .structName }}()
	ms.CopyTo(ms2)
	assert.Equal(t, ms.AsRaw(), ms2.AsRaw())
	assert.Panics(t, func() { ms2.CopyTo(ms) })

	assert.Panics(t, func() { ms.MoveTo(ms2) })
	assert.Panics(t, func() { ms2.MoveTo(ms) })
}

func Test{{ .structName }}Append(t *testing.T) {
	ms := New{{ .structName }}()
	ms.FromRaw([]{{ .itemType }}{ {{ .testOrigVal }} })
	ms.Append({{ .testSetVal }}, {{ .testSetVal }})
	assert.Equal(t, 5, ms.Len())
	assert.Equal(t, {{ .itemType }}({{ .testSetVal }}), ms.At(4))
}

func Test{{ .structName }}EnsureCapacity(t *testing.T) {
	ms := New{{ .structName }}()
	ms.EnsureCapacity(4)
	assert.Equal(t, 4, cap(*ms.getOrig()))
	ms.EnsureCapacity(2)
	assert.Equal(t, 4, cap(*ms.getOrig()))
}`

const primitiveSliceInternalTemplate = `
type {{ .structName }} struct {
	orig *[]{{ .itemType }}
	state *State
}

func GetOrig{{ .structName }}(ms {{ .structName }}) *[]{{ .itemType }} {
	return ms.orig
}

func Get{{ .structName }}State(ms {{ .structName }}) *State {
	return ms.state
}

func New{{ .structName }}(orig *[]{{ .itemType }}, state *State) {{ .structName }} {
	return {{ .structName }}{orig: orig, state: state}
}

func FillTest{{ .structName }}(tv {{ .structName}}) {
}

func GenerateTest{{ .structName }}() {{ .structName }} {
	state := StateMutable
	var orig []{{ .itemType }} = nil

	return {{ .structName }}{&orig, &state}
}`

// primitiveSliceStruct generates a struct for a slice of primitive value elements. The structs are always generated
// in a way that they can be used as fields in structs from other packages (using the internal package).
type primitiveSliceStruct struct {
	structName  string
	packageName string
	itemType    string

	testOrigVal          string
	testInterfaceOrigVal []interface{}
	testSetVal           string
	testNewVal           string
}

func (iss *primitiveSliceStruct) getName() string {
	return iss.structName
}

func (iss *primitiveSliceStruct) getPackageName() string {
	return iss.packageName
}

func (iss *primitiveSliceStruct) generateStruct(sb *bytes.Buffer) {
	t := template.Must(template.New("primitiveSliceTemplate").Parse(primitiveSliceTemplate))
	if err := t.Execute(sb, iss.templateFields()); err != nil {
		panic(err)
	}
}

func (iss *primitiveSliceStruct) generateTests(sb *bytes.Buffer) {
	t := template.Must(template.New("immutableSliceTestTemplate").Parse(immutableSliceTestTemplate))
	if err := t.Execute(sb, iss.templateFields()); err != nil {
		panic(err)
	}
}

func (iss *primitiveSliceStruct) generateTestValueHelpers(*bytes.Buffer) {}

func (iss *primitiveSliceStruct) generateInternal(sb *bytes.Buffer) {
	t := template.Must(template.New("primitiveSliceInternalTemplate").Parse(primitiveSliceInternalTemplate))
	if err := t.Execute(sb, iss.templateFields()); err != nil {
		panic(err)
	}
}

func (iss *primitiveSliceStruct) templateFields() map[string]any {
	return map[string]any{
		"structName":           iss.structName,
		"itemType":             iss.itemType,
		"lowerStructName":      strings.ToLower(iss.structName[:1]) + iss.structName[1:],
		"testOrigVal":          iss.testOrigVal,
		"testInterfaceOrigVal": iss.testInterfaceOrigVal,
		"testSetVal":           iss.testSetVal,
		"testNewVal":           iss.testNewVal,
	}
}
