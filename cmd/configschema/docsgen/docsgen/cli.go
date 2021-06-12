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
	"strings"
	"text/template"

	"go.opentelemetry.io/collector/cmd/configschema/configschema"
	"go.opentelemetry.io/collector/component"
)

const mdFileName = "config.md"

// CLI is the entrypoint for this package's functionality. It handles command-
// line arguments for the docsgen executable and produces config documentation
// for the specified components.
func CLI(factories component.Factories, dr configschema.DirResolver) {
	tableTmpl, err := tableTemplate()
	if err != nil {
		panic(err)
	}

	handleCLI(factories, dr, tableTmpl, ioutil.WriteFile, os.Stdout, os.Args...)
}

func handleCLI(
	factories component.Factories,
	dr configschema.DirResolver,
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
		allComponents(dr, tableTmpl, factories, writeFile)
		return
	}

	singleComponent(dr, tableTmpl, factories, componentType, args[2], writeFile)
}

func printLines(wr io.Writer, lines ...string) {
	for _, line := range lines {
		_, _ = fmt.Fprintln(wr, line)
	}
}

func allComponents(
	dr configschema.DirResolver,
	tableTmpl *template.Template,
	factories component.Factories,
	writeFile writeFileFunc,
) {
	configs := configschema.GetAllCfgInfos(factories)
	for _, cfg := range configs {
		writeConfigDoc(tableTmpl, dr, cfg, writeFile)
	}
}

func singleComponent(
	dr configschema.DirResolver,
	tableTmpl *template.Template,
	factories component.Factories,
	componentType, componentName string,
	writeFile writeFileFunc,
) {
	cfg, err := configschema.GetCfgInfo(factories, componentType, componentName)
	if err != nil {
		panic(err)
	}

	writeConfigDoc(tableTmpl, dr, cfg, writeFile)
}

type writeFileFunc func(filename string, data []byte, perm os.FileMode) error

func writeConfigDoc(
	tableTmpl *template.Template,
	dr configschema.DirResolver,
	ci configschema.CfgInfo,
	writeFile writeFileFunc,
) {
	v := reflect.ValueOf(ci.CfgInstance)
	f := configschema.ReadFields(v, dr)

	f.Type = stripPrefix(f.Type)

	mdBytes := renderHeader(string(ci.Type), ci.Group, f.Doc)
	tableBytes, err := renderTable(tableTmpl, f)
	if err != nil {
		panic(err)
	}
	mdBytes = append(mdBytes, tableBytes...)

	if hasTimeDuration(f) {
		mdBytes = append(mdBytes, durationBlock...)
	}

	dir := dr.PackageDir(v.Type().Elem())
	err = writeFile(path.Join(dir, mdFileName), mdBytes, 0644)
	if err != nil {
		panic(err)
	}
}

func stripPrefix(name string) string {
	idx := strings.Index(name, ".")
	return name[idx+1:]
}
