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

package semconv

import (
	"os"
	"path/filepath"
	"testing"

	version "github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
)

func TestAllSemConvFilesAreCrated(t *testing.T) {
	// Files that have to be present in each semconv package
	var expectedFiles = []string{"generated_resource.go", "generated_trace.go", "schema.go", "nonstandard.go"}

	files, err := os.ReadDir(".")
	assert.NoError(t, err)

	constraints, err := version.NewConstraint("> v1.16.0")
	assert.NoError(t, err)

	for _, f := range files {
		if !f.IsDir() {
			continue
		}

		ver, err := version.NewVersion(f.Name())
		assert.NoError(t, err)

		expected := make([]string, len(expectedFiles))
		copy(expected, expectedFiles)

		if constraints.Check(ver) {
			expected[len(expected)-1] = "generated_event.go"
		}

		for _, ef := range expected {
			assert.FileExists(t, filepath.Join(".", f.Name(), ef))
		}
	}
}
