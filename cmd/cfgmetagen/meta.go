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

package main

import (
	"io/ioutil"
	"path"
	"reflect"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type field struct {
	Name    string      `yaml:",omitempty"`
	Type    string      `yaml:",omitempty"`
	Kind    string      `yaml:",omitempty"`
	Default interface{} `yaml:",omitempty"`
	Doc     string      `yaml:",omitempty"`
	Meta    string      `yaml:",omitempty"`
	Fields  []*field    `yaml:",omitempty"`
}

// genMeta creates a `cfgMeta.yaml` file in the directory of the passed-in
// config instance. The yaml file contains the recursive field names, types,
// tags, comments, and default values for the config struct.
func genMeta(cfg interface{}) {
	cfgVal := reflect.ValueOf(cfg)
	cfgType := cfgVal.Type()
	field := &field{
		Type: cfgType.String(),
	}
	buildField(cfgVal, field)
	marshaled, err := yaml.Marshal(field)
	if err != nil {
		panic(err)
	}
	pth := typePath(cfgType.Elem())
	err = ioutil.WriteFile(path.Join(pth, "cfgMeta.yaml"), marshaled, 0600)
	if err != nil {
		panic(err)
	}
}

func buildField(v reflect.Value, f *field) {
	if v.Kind() == reflect.Ptr {
		buildField(v.Elem(), f)
	}
	if v.Kind() != reflect.Struct {
		return
	}
	comments := fieldCommentsForStruct(v)
	for i := 0; i < v.NumField(); i++ {
		structField := v.Type().Field(i)
		yamlName, meta := splitTag(structField.Tag)
		if yamlName == "-" {
			continue
		}
		fv := v.Field(i)
		next := f
		if meta != "squash" {
			doc := comments[structField.Name]
			name := yamlName
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
				Meta: meta,
			}
			f.Fields = append(f.Fields, next)
		}
		handleKinds(fv, next)
	}
}

func handleKinds(v reflect.Value, f *field) {
	switch v.Kind() {
	case reflect.Struct:
		buildField(v, f)
	case reflect.Ptr:
		if v.IsNil() {
			buildField(reflect.New(v.Type().Elem()), f)
		} else {
			buildField(v.Addr(), f)
		}
	case reflect.Slice:
		e := v.Type().Elem()
		if e.Kind() == reflect.Struct {
			buildField(reflect.New(e), f)
		} else if e.Kind() == reflect.Ptr {
			buildField(reflect.New(e.Elem()), f)
		}
	case reflect.String:
		if v.String() != "" {
			f.Default = v.String()
		}
	case reflect.Bool:
		f.Default = v.Bool()
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

func splitTag(s reflect.StructTag) (string, string) {
	ms := strings.TrimPrefix(string(s), "mapstructure:")
	trimmed := strings.Trim(ms, `"`)
	split := strings.Split(trimmed, ",")
	meta := ""
	if len(split) == 2 {
		meta = strings.TrimSpace(split[1])
	}
	return split[0], meta
}
