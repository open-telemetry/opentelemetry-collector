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
)

const (
	// Absolute root path of the project
	projectPath = "project-path"
	// Relative path where imports all default components
	relativeDefaultComponentsPath = "component-rel-path"
	// The project Go Module name
	projectGoModule = "module-name"
)

// The main verifies if README.md and proper documentations for the enabled default components
// are existed in OpenTelemetry core and contrib repository.
// Usage in the core repo:
// checkdoc --project-path path/to/project \
//			--component-rel-path service/defaultcomponents/defaults.go \
//			--module-name go.opentelemetry.io/collector
//
// Usage in the contrib repo:
// checkdoc --project-path path/to/project \
//			--component-rel-path cmd/otelcontrib/components.go \
//			--module-name github.com/open-telemetry/opentelemetry-collector-contrib
//
func main() {
	projectPath := flag.String(projectPath, "", "specify the project path")
	componentPath := flag.String(relativeDefaultComponentsPath, "", "specify the relative component path")
	moduleName := flag.String(projectGoModule, "", "specify the project go module")

	flag.Parse()

	err := checkDocs(
		*projectPath,
		*componentPath,
		*moduleName,
	)

	if err != nil {
		panic(err)
	}
}
