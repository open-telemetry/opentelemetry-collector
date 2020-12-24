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

package schemagen

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"time"

	"github.com/fatih/structtag"
	"gopkg.in/yaml.v2"
)

type field struct {
	Name    string      `yaml:",omitempty"`
	Type    string      `yaml:",omitempty"`
	Kind    string      `yaml:",omitempty"`
	Default interface{} `yaml:",omitempty"`
	Doc     string      `yaml:",omitempty"`
	Fields  []*field    `yaml:",omitempty"`
}

// createSchemaFile creates a `cfg-schema.yaml` file in the directory of the passed-in
// config instance. The yaml file contains the recursive field names, types,
// comments, and default values for the config struct.
func createSchemaFile(cfg interface{}, env env) {
	v := reflect.ValueOf(cfg)
	f := topLevelField(v, env)
	yamlFilename := env.yamlFilename(v.Type().Elem(), env)
	writeMarshaled(f, yamlFilename)
}

func topLevelField(v reflect.Value, env env) *field {
	cfgType := v.Type()
	field := &field{
		Type: cfgType.String(),
	}
	refl(field, v, env)
	return field
}

func refl(f *field, v reflect.Value, env env) {
	if v.Kind() == reflect.Ptr {
		refl(f, v.Elem(), env)
	}
	if v.Kind() != reflect.Struct {
		return
	}
	comments := commentsForStruct(v, env)
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
			doc := comments[structField.Name]
			name := tagName
			if name == "" {
				name = strings.ToLower(structField.Name)
			}
			kindStr := fv.Kind().String()
			typeStr := fv.Type().String()
			if typeStr == kindStr {
				typeStr = "" // omit if redundant
			}
			next = &field{
				Name: name,
				Type: typeStr,
				Kind: kindStr,
				Doc:  doc,
			}
			f.Fields = append(f.Fields, next)
		}
		handleKinds(fv, next, env)
	}
}

func handleKinds(v reflect.Value, f *field, env env) {
	switch v.Kind() {
	case reflect.Struct:
		refl(f, v, env)
	case reflect.Ptr:
		if v.IsNil() {
			refl(f, reflect.New(v.Type().Elem()), env)
		} else {
			refl(f, v.Elem(), env)
		}
	case reflect.Slice:
		e := v.Type().Elem()
		if e.Kind() == reflect.Struct {
			refl(f, reflect.New(e), env)
		} else if e.Kind() == reflect.Ptr {
			refl(f, reflect.New(e.Elem()), env)
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

func writeMarshaled(field *field, filename string) {
	marshaled, err := yaml.Marshal(field)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(filename, marshaled, 0600)
	if err != nil {
		panic(err)
	}
}
