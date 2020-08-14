/*
 * Copyright The OpenTelemetry Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/non-standard/validators"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"gopkg.in/yaml.v2"
)

type metricType string

// metricTypeMapping maps yaml name to code type name.
var metricTypeMapping = map[metricType]string{
	"int64":       "pdata.MetricTypeInt64",
	"double":      "pdata.MetricTypeDouble",
	"int64 mono":  "pdata.MetricTypeMonotonicInt64",
	"double mono": "pdata.MetricTypeMonotonicDouble",
	"histogram":   "pdata.MetricTypeHistogram",
	"summary":     "pdata.MetricTypeSummary",
}

func (mt metricType) Render() (string, error) {
	if codeName, ok := metricTypeMapping[mt]; ok {
		return codeName, nil
	}
	return "", fmt.Errorf("unknown metric type %q", mt)
}

type metricName string

func (mn metricName) Render() (string, error) {
	return formatVar(string(mn), true)
}

type labelName string

func (mn labelName) Render() (string, error) {
	return formatVar(string(mn), true)
}

type metric struct {
	Description string     `validate:"required,notblank"`
	Unit        string     `validate:"oneof=s By"`
	Type        metricType `validate:"metrictype"`
	Labels      []labelName
}

type label struct {
	// Description describes the purpose of the label.
	Description string `validate:"notblank"`
	// Value can optionally specify the value this label will have.
	// For example, the label may have the identifier `MemState` to its
	// value may be `state` when used.
	Value string
	// Enum can optionally describe the set of values to which the label can belong.
	Enum []string
}

type metadata struct {
	Name    string                `validate:"notblank"`
	Labels  map[labelName]label   `validate:"dive"`
	Metrics map[metricName]metric `validate:"dive"`
}

type templateContext struct {
	metadata
	Package string
}

func loadMetadata(yml string) metadata {
	ymlData, err := ioutil.ReadFile(yml)
	if err != nil {
		log.Fatalf("unable to read file %v: %v", yml, err)
	}

	var out metadata

	// Unmarshal metadata.
	if err := yaml.Unmarshal(ymlData, &out); err != nil {
		log.Fatalf("unable to unmarshal yaml from %v: %v", yml, err)
	}

	// Perform validation of unmarshaled structs.
	v := validator.New()
	if err := v.RegisterValidation("notblank", validators.NotBlank); err != nil {
		log.Fatalf("failed registering notblank validator: %v", err)
	}
	if err := v.RegisterValidation("metrictype", func(fl validator.FieldLevel) bool {
		return metricTypeMapping[metricType(fl.Field().String())] != ""
	}); err != nil {
		log.Fatalf("failed registering metric-type validator: %v", err)
	}

	// Provides better validation error messages.
	enLocale := en.New()
	uni := ut.New(enLocale, enLocale)

	tr, ok := uni.GetTranslator("en")
	if !ok {
		log.Fatal("unable to lookup en translator")
	}

	if err := en_translations.RegisterDefaultTranslations(v, tr); err != nil {
		log.Fatalf("failed registering translations: %v", err)
	}

	if err := tr.Add("nosuchlabel", "{0} {1} no such label ok??", true); err != nil {
		log.Fatalf("adding translation failed: %v", err)
	}

	if err := v.RegisterTranslation("nosuchlabel", tr, func(ut ut.Translator) error {
		return ut.Add("nosuchlabel", "unknown label value", true) // see universal-translator for details
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("nosuchlabel", fe.Field())
		return t
	}); err != nil {
		log.Fatalf("failed registering nosuchlabel: %v", err)
	}

	v.RegisterStructValidation(metricValidation, metric{})

	if err := v.Struct(&out); err != nil {
		if verr, ok := err.(validator.ValidationErrors); ok {
			m := verr.Translate(tr)
			log.Print("error validating struct:")
			for k, v := range m {
				log.Printf("\t%v: %v", k, v)
			}
			os.Exit(1)
		}
		log.Fatalf("unknown validation error: %v", err)
	}

	return out
}

// metricValidation validates metric structs.
func metricValidation(sl validator.StructLevel) {
	// Make sure that the labels are valid.
	md := sl.Top().Interface().(*metadata)
	cur := sl.Current().Interface().(metric)

	for _, l := range cur.Labels {
		if _, ok := md.Labels[l]; !ok {
			sl.ReportError(cur.Labels, fmt.Sprintf("Labels[%s]", string(l)), "Labels", "nosuchlabel", "")
		}
	}
}
