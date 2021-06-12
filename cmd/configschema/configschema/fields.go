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

package configschema

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/fatih/structtag"
)

// Field holds attributes and subfields of a config struct.
type Field struct {
	Name    string      `yaml:",omitempty"`
	Type    string      `yaml:",omitempty"`
	Kind    string      `yaml:",omitempty"`
	Default interface{} `yaml:",omitempty"`
	Doc     string      `yaml:",omitempty"`
	Fields  []*Field    `yaml:",omitempty"`
}

// ReadFields accepts both a config struct's Value, as well as a DirResolver,
// and returns a Field pointer for the top level struct as well as all of its
// recursive subfields.
func ReadFields(v reflect.Value, dr DirResolver) *Field {
	cfgType := v.Type()
	field := &Field{
		Type: cfgType.String(),
	}
	refl(field, v, dr)
	return field
}

func refl(f *Field, v reflect.Value, dr DirResolver) {
	if v.Kind() == reflect.Ptr {
		refl(f, v.Elem(), dr)
	}
	if v.Kind() != reflect.Struct {
		return
	}
	comments := commentsForStruct(v, dr)

	// we also check if f.Doc hasn't already been written, thus preventing a
	// squashed type with struct comments from overwriting the containing struct's
	// comments
	if sc, ok := comments["_struct"]; ok && f.Doc == "" {
		f.Doc = sc
	}

	for i := 0; i < v.NumField(); i++ {
		structField := v.Type().Field(i)
		tagName, options, err := mapstructure(structField.Tag)
		if err != nil {
			fmt.Printf("error parsing mapstructure tag for field %v: %q", structField, err.Error())
			// not fatal, can keep going
		}
		if tagName == "-" {
			continue
		}
		fv := v.Field(i)
		next := f
		if !containsSquash(options) {
			name := tagName
			if name == "" {
				name = strings.ToLower(structField.Name)
			}
			kindStr := fv.Kind().String()
			typeStr := fv.Type().String()
			if typeStr == kindStr {
				typeStr = "" // omit if redundant
			}
			next = &Field{
				Name: name,
				Type: typeStr,
				Kind: kindStr,
				Doc:  comments[structField.Name],
			}
			f.Fields = append(f.Fields, next)
		}
		handleKind(fv, next, dr)
	}
}

func handleKind(v reflect.Value, f *Field, dr DirResolver) {
	switch v.Kind() {
	case reflect.Struct:
		refl(f, v, dr)
	case reflect.Ptr:
		if v.IsNil() {
			refl(f, reflect.New(v.Type().Elem()), dr)
		} else {
			refl(f, v.Elem(), dr)
		}
	case reflect.Slice:
		e := v.Type().Elem()
		if e.Kind() == reflect.Struct {
			refl(f, reflect.New(e), dr)
		} else if e.Kind() == reflect.Ptr {
			refl(f, reflect.New(e.Elem()), dr)
		}
	case reflect.String:
		if v.String() != "" {
			f.Default = v.String()
		}
	case reflect.Bool:
		if v.Bool() {
			f.Default = v.Bool()
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.Int() != 0 {
			if v.Type() == reflect.TypeOf(time.Duration(0)) {
				f.Default = time.Duration(v.Int()).String()
			} else {
				f.Default = v.Int()
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v.Uint() != 0 {
			f.Default = v.Uint()
		}
	}
}

func mapstructure(st reflect.StructTag) (string, []string, error) {
	tag := string(st)
	if tag == "" {
		return "", nil, nil
	}
	tags, err := structtag.Parse(tag)
	if err != nil {
		return "", nil, err
	}
	ms, err := tags.Get("mapstructure")
	if err != nil {
		return "", nil, err
	}
	return ms.Name, ms.Options, nil
}

func containsSquash(options []string) bool {
	for _, option := range options {
		if option == "squash" {
			return true
		}
	}
	return false
}
