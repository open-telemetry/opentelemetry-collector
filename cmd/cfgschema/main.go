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
	"flag"
	"fmt"

	cfgschema "go.opentelemetry.io/collector/cmd/cfgschema/internal"
)

func main() {
	prepUsage()

	e, componentType, componentName := parseArgs()

	switch {
	case componentType == "all":
		cfgschema.CreateAllCfgSchemaFiles(e)
	case componentType != "" && componentName != "":
		cfgschema.CreateSingleCfgSchema(componentType, componentName, e)
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

func parseArgs() (cfgschema.Env, string, string) {
	e := cfgschema.Env{}
	flag.StringVar(&e.SrcRoot, "s", cfgschema.DefaultSrcRoot, "collector source root")
	flag.StringVar(&e.ModuleName, "m", cfgschema.DefaultModule, "module name")
	flag.Parse()
	componentType := flag.Arg(0)
	componentName := flag.Arg(1)
	return e, componentType, componentName
}
