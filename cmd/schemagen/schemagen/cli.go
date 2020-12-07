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
	"flag"
	"fmt"

	"go.opentelemetry.io/collector/component"
)

func CLI(c component.Factories) {
	prepUsage()

	e, componentType, componentName := parseArgs()
	e.YamlFilename = YamlFilename

	switch {
	case componentType == "all":
		CreateAllSchemaFiles(c, e)
	case componentType != "" && componentName != "":
		CreateSingleSchemaFile(
			c,
			componentType,
			componentName,
			e,
		)
	default:
		flag.Usage()
	}
}

func prepUsage() {
	const usage = `cfgschema all
cfgschema <componentType> <componentName>

options
`
	flag.Usage = func() {
		_, _ = fmt.Fprint(flag.CommandLine.Output(), usage)
		flag.PrintDefaults()
	}
}

func parseArgs() (Env, string, string) {
	e := Env{}
	flag.StringVar(&e.SrcRoot, "s", DefaultSrcRoot, "collector source root")
	flag.StringVar(&e.ModuleName, "m", DefaultModule, "module name")
	flag.Parse()
	componentType := flag.Arg(0)
	componentName := flag.Arg(1)
	return e, componentType, componentName
}
