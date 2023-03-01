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

// New{{ .structName }} creates a new empty {{ .structName }}.
func New{{ .structName }}() {{ .structName }} {
	orig := []{{ .itemType }}(nil)
	return {{ .structName }}(internal.New{{ .structName }}(&orig))
}

// AsRaw returns a copy of the []{{ .itemType }} slice.
func (ms {{ .structName }}) AsRaw() []{{ .itemType }} {
	return copy{{ .structName }}(nil, *ms.getOrig())
}

// FromRaw copies raw []{{ .itemType }} into the slice {{ .structName }}.
func (ms {{ .structName }}) FromRaw(val []{{ .itemType }}) {
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
	(*ms.getOrig())[i] = val
}

// EnsureCapacity ensures {{ .structName }} has at least the specified capacity.
// 1. If the newCap <= cap, then is no change in capacity.
// 2. If the newCap > cap, then the slice capacity will be expanded to the provided value which will be equivalent of:
//	buf := make([]{{ .itemType }}, len({{ .lowerStructName }}), newCap)
//	copy(buf, {{ .lowerStructName }})
//	{{ .lowerStructName }} = buf
func (ms {{ .structName }}) EnsureCapacity(newCap int) {
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
	*ms.getOrig() = append(*ms.getOrig(), elms...)
}

// MoveTo moves all elements from the current slice overriding the destination and 
// resetting the current instance to its zero value.
func (ms {{ .structName }}) MoveTo(dest {{ .structName }}) {
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = nil
}

// CopyTo copies all elements from the current slice overriding the destination.
func (ms {{ .structName }}) CopyTo(dest {{ .structName }}) {
	*dest.getOrig() = copy{{ .structName }}(*dest.getOrig(), *ms.getOrig())
}

func copy{{ .structName }}(dst, src []{{ .itemType }}) []{{ .itemType }} {
	dst = dst[:0]
	return append(dst, src...)
}`

const immutableSliceTestTemplate = `func TestNew{{ .structName }}(t *testing.T) {
	ms := New{{ .structName }}()
	assert.Equal(t, 0, ms.Len())
	ms.FromRaw([]{{ .itemType }}{1, 2, 3})
	assert.Equal(t, 3, ms.Len())
	assert.Equal(t, []{{ .itemType }}{1, 2, 3}, ms.AsRaw())
	ms.SetAt(1, {{ .itemType }}(5))
	assert.Equal(t, []{{ .itemType }}{1, 5, 3}, ms.AsRaw())
	ms.FromRaw([]{{ .itemType }}{3})
	assert.Equal(t, 1, ms.Len())
	assert.Equal(t, {{ .itemType }}(3), ms.At(0))
	
	cp := New{{ .structName }}()
	ms.CopyTo(cp)
	ms.SetAt(0, {{ .itemType }}(2))
	assert.Equal(t, {{ .itemType }}(2), ms.At(0))
	assert.Equal(t, {{ .itemType }}(3), cp.At(0))
	ms.CopyTo(cp)
	assert.Equal(t, {{ .itemType }}(2), cp.At(0))
	
	mv := New{{ .structName }}()
	ms.MoveTo(mv)
	assert.Equal(t, 0, ms.Len())
	assert.Equal(t, 1, mv.Len())
	assert.Equal(t, {{ .itemType }}(2), mv.At(0))
	ms.FromRaw([]{{ .itemType }}{1, 2, 3})
	ms.MoveTo(mv)
	assert.Equal(t, 3, mv.Len())
	assert.Equal(t, {{ .itemType }}(1), mv.At(0))
}

func Test{{ .structName }}Append(t *testing.T) {
	ms := New{{ .structName }}()
	ms.FromRaw([]{{ .itemType }}{1, 2, 3})
	ms.Append(4, 5)
	assert.Equal(t, 5, ms.Len())
	assert.Equal(t, {{ .itemType }}(5), ms.At(4))
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
}

func GetOrig{{ .structName }}(ms {{ .structName }}) *[]{{ .itemType }} {
	return ms.orig
}

func New{{ .structName }}(orig *[]{{ .itemType }}) {{ .structName }} {
	return {{ .structName }}{orig: orig}
}`

// primitiveSliceStruct generates a struct for a slice of primitive value elements. The structs are always generated
// in a way that they can be used as fields in structs from other packages (using the internal package).
type primitiveSliceStruct struct {
	structName  string
	packageName string
	itemType    string
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
		"structName":      iss.structName,
		"itemType":        iss.itemType,
		"lowerStructName": strings.ToLower(iss.structName[:1]) + iss.structName[1:],
	}
}
