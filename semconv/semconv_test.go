// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package semconv

import (
	"os"
	"path/filepath"
	"testing"

	version "github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllSemConvFilesAreCrated(t *testing.T) {
	// Files that have to be present in each semconv package
	var expectedFiles = []string{"generated_resource.go", "generated_trace.go", "schema.go", "nonstandard.go"}

	files, err := os.ReadDir(".")
	require.NoError(t, err)

	constraints, err := version.NewConstraint("> v1.16.0")
	require.NoError(t, err)

	attrgroupconstraints, err := version.NewConstraint("> v1.22.0")
	require.NoError(t, err)

	for _, f := range files {
		if !f.IsDir() || f.Name() == "weaver" {
			continue
		}

		ver, err := version.NewVersion(f.Name())
		require.NoError(t, err)

		expected := make([]string, len(expectedFiles))
		copy(expected, expectedFiles)

		if constraints.Check(ver) {
			expected[len(expected)-1] = "generated_event.go"
		}

		if attrgroupconstraints.Check(ver) {
			assert.FileExists(t, filepath.Join(".", f.Name(), "generated_attribute_group.go"))
		}

		for _, ef := range expected {
			assert.FileExists(t, filepath.Join(".", f.Name(), ef))
		}
	}
}
