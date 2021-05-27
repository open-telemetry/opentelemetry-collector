// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/non-standard/validators"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"gopkg.in/yaml.v2"
)

type metricName string

func (mn metricName) Render() (string, error) {
	return formatIdentifier(string(mn), true)
}

type labelName string

func (mn labelName) Render() (string, error) {
	return formatIdentifier(string(mn), true)
}

type metric struct {
	// Description of the metric.
	Description string `validate:"required,notblank"`
	// Unit of the metric.
	Unit string `yaml:"unit"`

	// Raw data that is used to set Data interface below.
	YmlData *ymlMetricData `yaml:"data" validate:"required"`
	// Date is set to generic metric data interface after validating.
	Data MetricData `yaml:"-"`

	// Labels is the list of labels that the metric emits.
	Labels []labelName
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
	// Name of the component.
	Name string `validate:"notblank"`
	// Labels emitted by one or more metrics.
	Labels map[labelName]label `validate:"dive"`
	// Metrics that can be emitted by the component.
	Metrics map[metricName]metric `validate:"dive"`
}

type templateContext struct {
	metadata
	// Package name for generated code.
	Package string
}

func loadMetadata(ymlData []byte) (metadata, error) {
	var out metadata

	// Unmarshal metadata.
	if err := yaml.Unmarshal(ymlData, &out); err != nil {
		return metadata{}, fmt.Errorf("unable to unmarshal yaml: %v", err)
	}

	// Validate metadata.
	if err := validateMetadata(out); err != nil {
		return metadata{}, err
	}

	return out, nil
}

func validateMetadata(out metadata) error {
	v := validator.New()
	if err := v.RegisterValidation("notblank", validators.NotBlank); err != nil {
		return fmt.Errorf("failed registering notblank validator: %v", err)
	}

	// Provides better validation error messages.
	enLocale := en.New()
	uni := ut.New(enLocale, enLocale)

	tr, ok := uni.GetTranslator("en")
	if !ok {
		return errors.New("unable to lookup en translator")
	}

	if err := en_translations.RegisterDefaultTranslations(v, tr); err != nil {
		return fmt.Errorf("failed registering translations: %v", err)
	}

	if err := v.RegisterTranslation("nosuchlabel", tr, func(ut ut.Translator) error {
		return ut.Add("nosuchlabel", "unknown label value", true) // see universal-translator for details
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("nosuchlabel", fe.Field())
		return t
	}); err != nil {
		return fmt.Errorf("failed registering nosuchlabel: %v", err)
	}

	v.RegisterStructValidation(metricValidation, metric{})

	if err := v.Struct(&out); err != nil {
		if verr, ok := err.(validator.ValidationErrors); ok {
			m := verr.Translate(tr)
			buf := strings.Builder{}
			buf.WriteString("error validating struct:\n")
			for k, v := range m {
				buf.WriteString(fmt.Sprintf("\t%v: %v\n", k, v))
			}
			return errors.New(buf.String())
		}
		return fmt.Errorf("unknown validation error: %v", err)
	}

	// Set metric data interface.
	for k, v := range out.Metrics {
		v.Data = v.YmlData.MetricData
		v.YmlData = nil
		out.Metrics[k] = v
	}

	return nil
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
