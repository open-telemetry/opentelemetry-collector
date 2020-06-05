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

package defaultcomponents

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Import importPrefixesToCheck to check for readmes.
var importPrefixesToCheck = []string{
	"go.opentelemetry.io/collector/extension",
	"go.opentelemetry.io/collector/receiver",
	"go.opentelemetry.io/collector/processor",
	"go.opentelemetry.io/collector/exporter",
}

const (
	defaultComponents = "defaults.go"
	readme            = "README.md"
)

// TestComponentDocs verifies existence of READMEs for components specified as
// default components in the collector. Looking for default components being enabled
// in the collector gives a reasonable measure of the components that need to be
// documented. Note, that for this test to work, the underlying assumption is
// the imports in "service/defaultcomponents/defaults.go" are indicative
// of components that require documentation.
func TestComponentDocs(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err, "failed to get working directory")

	defaultComponentsFilePath := filepath.Join(wd, defaultComponents)
	_, err = os.Stat(defaultComponentsFilePath)
	require.NoError(t, err, "failed to load file %s", defaultComponentsFilePath)

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, defaultComponentsFilePath, nil, parser.ImportsOnly)
	if err != nil {
		t.Errorf("failed to load imports: %v", err)
	}

	// Absolute path to the project root directory
	projectPath := filepath.Join(wd, "../../")

	for _, i := range f.Imports {
		importPath := strings.Trim(i.Path.Value, `"`)

		if isComponentImport(importPath) {
			relativeComponentPath := strings.Replace(importPath, "go.opentelemetry.io/collector/", "", 1)
			readmePath := filepath.Join(projectPath, relativeComponentPath, readme)
			_, err := os.Stat(readmePath)
			require.NoError(t, err, "README does not exist at %s, add one", readmePath)
		}
	}
}

// isComponentImport returns true if the import corresponds to
// a Otel component, i.e. an extension, exporter, processor or
// a receiver.
func isComponentImport(importStr string) bool {
	// Imports representative of components are of the following types
	// "go.opentelemetry.io/collector/processor/samplingprocessor/probabilisticsamplerprocessor"
	// "go.opentelemetry.io/collector/processor/batchprocessor"
	// where these imports can be of exporters, extensions, receivers or
	// processors.
	for _, prefix := range importPrefixesToCheck {
		if strings.HasPrefix(importStr, prefix) {
			return true
		}
	}

	return false
}
