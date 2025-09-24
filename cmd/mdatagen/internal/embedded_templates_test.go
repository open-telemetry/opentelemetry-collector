// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"io/fs"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureTemplatesLoaded(t *testing.T) {
	t.Parallel()

	const (
		rootDir = "templates"
	)

	var (
		templateFiles = map[string]struct{}{
			path.Join(rootDir, "component_test.go.tmpl"):       {},
			path.Join(rootDir, "documentation.md.tmpl"):        {},
			path.Join(rootDir, "metrics.go.tmpl"):              {},
			path.Join(rootDir, "metrics_test.go.tmpl"):         {},
			path.Join(rootDir, "logs.go.tmpl"):                 {},
			path.Join(rootDir, "logs_test.go.tmpl"):            {},
			path.Join(rootDir, "resource.go.tmpl"):             {},
			path.Join(rootDir, "resource_test.go.tmpl"):        {},
			path.Join(rootDir, "config.go.tmpl"):               {},
			path.Join(rootDir, "config_test.go.tmpl"):          {},
			path.Join(rootDir, "package_test.go.tmpl"):         {},
			path.Join(rootDir, "readme.md.tmpl"):               {},
			path.Join(rootDir, "status.go.tmpl"):               {},
			path.Join(rootDir, "telemetry.go.tmpl"):            {},
			path.Join(rootDir, "telemetry_test.go.tmpl"):       {},
			path.Join(rootDir, "testdata", "config.yaml.tmpl"): {},
			path.Join(rootDir, "telemetrytest.go.tmpl"):        {},
			path.Join(rootDir, "telemetrytest_test.go.tmpl"):   {},
			path.Join(rootDir, "helper.tmpl"):                  {},
		}
		count = 0
	)
	require.NoError(t, fs.WalkDir(TemplateFS, ".", func(path string, d fs.DirEntry, _ error) error {
		if d != nil && d.IsDir() {
			return nil
		}
		count++
		assert.Contains(t, templateFiles, path)
		return nil
	}))
	assert.Equal(t, len(templateFiles), count, "Must match the expected number of calls")
}
