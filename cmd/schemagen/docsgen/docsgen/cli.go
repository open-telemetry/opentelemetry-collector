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

package docsgen

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"text/template"

	"go.opentelemetry.io/collector/cmd/schemagen/configschema"
	"go.opentelemetry.io/collector/component"
)

const (
	mdFileName  = "config.md"
	templateDir = "cmd/schemagen/docsgen/docsgen/templates/"
)

// CLI is the entrypoint for this package's functionality. It handles command-
// line arguments for the docsgen executable and produces config documentation
// for the specified components.
func CLI(factories component.Factories) {
	headerTmpl, err := headerTemplate(templateDir)
	if err != nil {
		panic(err)
	}

	tableTmpl, err := tableTemplate(templateDir)
	if err != nil {
		panic(err)
	}

	dr := configschema.NewDefaultDirResolver()
	handleCLI(factories, dr, headerTmpl, tableTmpl, ioutil.WriteFile, os.Stdout, os.Args...)
}

func handleCLI(
	factories component.Factories,
	dr configschema.DirResolver,
	headerTmpl *template.Template,
	tableTmpl *template.Template,
	writeFile writeFileFunc,
	wr io.Writer,
	args ...string,
) {
	if !(len(args) == 2 || len(args) == 3) {
		printLines(wr, "usage:", "docsgen all", "docsgen component-type component-name")
		return
	}

	componentType := args[1]
	if componentType == "all" {
		allComponents(dr, headerTmpl, tableTmpl, factories, writeFile)
		return
	}

	singleComponent(dr, headerTmpl, tableTmpl, factories, componentType, args[2], writeFile)
}

func printLines(wr io.Writer, lines ...string) {
	for _, line := range lines {
		_, _ = fmt.Fprintln(wr, line)
	}
}

func allComponents(
	dr configschema.DirResolver,
	headerTmpl *template.Template,
	tableTmpl *template.Template,
	factories component.Factories,
	writeFile writeFileFunc,
) {
	configs := configschema.GetAllConfigs(factories)
	for _, cfg := range configs {
		writeConfigDoc(headerTmpl, tableTmpl, dr, cfg, writeFile)
	}
}

func singleComponent(
	dr configschema.DirResolver,
	headerTmpl, tableTmpl *template.Template,
	factories component.Factories,
	componentType, componentName string,
	writeFile writeFileFunc,
) {
	cfg, err := configschema.GetConfig(factories, componentType, componentName)
	if err != nil {
		panic(err)
	}

	writeConfigDoc(headerTmpl, tableTmpl, dr, cfg, writeFile)
}

type writeFileFunc func(filename string, data []byte, perm os.FileMode) error

func writeConfigDoc(
	headerTmpl *template.Template,
	tableTmpl *template.Template,
	dr configschema.DirResolver,
	config interface{},
	writeFile writeFileFunc,
) {
	v := reflect.ValueOf(config)
	f := configschema.ReadFields(v, dr)

	headerBytes, err := renderHeader(headerTmpl, f)
	if err != nil {
		panic(err)
	}

	tableBytes, err := renderTable(tableTmpl, f)
	if err != nil {
		panic(err)
	}

	dir := dr.PackageDir(v.Type().Elem())
	err = writeFile(path.Join(dir, mdFileName), append(headerBytes, tableBytes...), 0644)
	if err != nil {
		panic(err)
	}
}
