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
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"text/template"
)

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
				"publicVar": func(s string) (string, error) {
					return formatVar(s, true)
				},
				"privateVar": func(s string) (string, error) {
					return formatVar(s, false)
				},
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
