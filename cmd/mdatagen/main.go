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
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

func main() {
	flag.Parse()
	yml := flag.Arg(0)
	if err := run(yml); err != nil {
		log.Fatal(err)
	}
}

func run(ymlPath string) error {
	if ymlPath == "" {
		return errors.New("argument must be metadata.yaml file")
	}

	ymlDir := path.Dir(ymlPath)

	ymlData, err := ioutil.ReadFile(filepath.Clean(ymlPath))
	if err != nil {
		return fmt.Errorf("unable to read file %v: %v", ymlPath, err)
	}

	md, err := loadMetadata(ymlData)
	if err != nil {
		return fmt.Errorf("failed loading %v: %v", ymlPath, err)
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return errors.New("unable to determine filename")
	}
	thisDir := path.Dir(filename)

	if err = generateMetrics(ymlDir, thisDir, md); err != nil {
		return err
	}
	return generateDocumentation(ymlDir, thisDir, md)
}

func generateMetrics(ymlDir string, thisDir string, md metadata) error {
	tmpl := template.Must(
		template.
			New("metrics.tmpl").
			Option("missingkey=error").
			Funcs(map[string]interface{}{
				"publicVar": func(s string) (string, error) {
					return formatIdentifier(s, true)
				},
			}).ParseFiles(path.Join(thisDir, "metrics.tmpl")))

	buf := bytes.Buffer{}

	if err := tmpl.Execute(&buf, templateContext{metadata: md, Package: "metadata"}); err != nil {
		return fmt.Errorf("failed executing template: %v", err)
	}

	formatted, err := format.Source(buf.Bytes())

	if err != nil {
		errstr := strings.Builder{}
		_, _ = fmt.Fprintf(&errstr, "failed formatting source: %v", err)
		errstr.WriteString("--- BEGIN SOURCE ---")
		errstr.Write(buf.Bytes())
		errstr.WriteString("--- END SOURCE ---")
		return errors.New(errstr.String())
	}

	outputDir := path.Join(ymlDir, "internal", "metadata")
	outputFile := path.Join(outputDir, "generated_metrics.go")
	if err := os.MkdirAll(outputDir, 0700); err != nil {
		return fmt.Errorf("unable to create output directory %q: %v", outputDir, err)
	}
	if err := ioutil.WriteFile(outputFile, formatted, 0600); err != nil {
		return fmt.Errorf("failed writing %q: %v", outputFile, err)
	}

	return nil
}

func generateDocumentation(ymlDir string, thisDir string, md metadata) error {
	tmpl := template.Must(
		template.
			New("documentation.tmpl").
			Option("missingkey=error").
			Funcs(map[string]interface{}{
				"publicVar": func(s string) (string, error) {
					return formatIdentifier(s, true)
				},
			}).ParseFiles(path.Join(thisDir, "documentation.tmpl")))

	buf := bytes.Buffer{}

	if err := tmpl.Execute(&buf, templateContext{metadata: md, Package: "metadata"}); err != nil {
		return fmt.Errorf("failed executing template: %v", err)
	}

	outputFile := path.Join(ymlDir, "documentation.md")
	if err := ioutil.WriteFile(outputFile, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("failed writing %q: %v", outputFile, err)
	}

	return nil
}
