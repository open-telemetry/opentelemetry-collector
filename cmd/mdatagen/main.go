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
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"text/template"

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

type metrics struct {
	Description string     `validate:"required,notblank"`
	Unit        string     `validate:"oneof=s bytes"`
	Type        metricType `validate:"metrictype"`
}

type labels struct {
	Description string `validate:"notblank"`
	Enum        []string
}

type metadata struct {
	Name    string                 `validate:"notblank"`
	Labels  map[labelName]labels   `validate:"dive"`
	Metrics map[metricName]metrics `validate:"dive"`
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
		log.Fatalf("failed registering metric-type validator: %v")
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

func main() {
	flag.Parse()
	yml := flag.Arg(0)
	if yml == "" {
		log.Fatal("argument must be metadata.yaml file")
	}

	metadata := loadMetadata(yml)

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("unable to determine filename")
	}

	thisDir := path.Dir(filename)
	tmpl := template.Must(
		template.
			New("metrics.tmpl").
			Option("missingkey=error").
			Funcs(map[string]interface{}{
				"formatVar": formatVar,
	}).ParseFiles(path.Join(thisDir, "metrics.tmpl")))
	buf := bytes.Buffer{}

	if err := tmpl.Execute(&buf, templateContext{
		metadata: metadata,
		Package:  "metadata",
	}); err != nil {
		log.Fatalf("failed executing template: %v", err)
	}

	formatted, err := format.Source(buf.Bytes())

	if err != nil {
		// TODO: Remove or conditionalize later (for debugging).
		log.Println("--- BEGIN SOURCE ---")
		log.Print(buf.String())
		log.Println("--- END SOURCE ---")
		log.Fatalf("failed formatting source: %v", err)
	}

	if _, err := os.Stdout.Write(formatted); err != nil {
		log.Fatalf("write failed: %v", err)
	}

	metadir := path.Dir(yml)
	outputDir := path.Join(metadir, "internal", "metadata")
	outputFile := path.Join(outputDir, "generated_metrics.go")
	if err := os.MkdirAll(outputDir, 0777); err != nil {
		log.Fatalf("unable to create output directory %q: %v", outputDir, err)
	}
	if err := ioutil.WriteFile(outputFile, formatted, 0666); err != nil {
		log.Fatalf("failed writing %q: %v", outputFile, err)
	}
}
