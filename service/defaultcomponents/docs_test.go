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

package defaultcomponents

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
)

const (
	relativeDefaultComponentsPath = "service/defaultcomponents/defaults.go"
	projectGoModule               = "go.opentelemetry.io/collector"
)

// TestComponentDocs verifies existence of READMEs for components specified as
// default components in the collector. Looking for default components being enabled
// in the collector gives a reasonable measure of the components that need to be
// documented. Note, that for this test to work, the underlying assumption is
// the imports in "service/defaultcomponents/defaults.go" are indicative
// of components that require documentation.
func TestComponentDocs(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err, "failed to get working directory: %v")

	// Absolute path to the project root directory
	projectPath := filepath.Join(wd, "../../")

	err = componenttest.CheckDocs(
		projectPath,
		relativeDefaultComponentsPath,
		projectGoModule,
	)
	require.NoError(t, err)
}
