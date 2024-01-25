// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"io/fs"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
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
			path.Join(rootDir, "resource.go.tmpl"):             {},
			path.Join(rootDir, "resource_test.go.tmpl"):        {},
			path.Join(rootDir, "config.go.tmpl"):               {},
			path.Join(rootDir, "config_test.go.tmpl"):          {},
			path.Join(rootDir, "readme.md.tmpl"):               {},
			path.Join(rootDir, "status.go.tmpl"):               {},
			path.Join(rootDir, "testdata", "config.yaml.tmpl"): {},
		}
		count = 0
	)
	assert.NoError(t, fs.WalkDir(templateFS, ".", func(path string, d fs.DirEntry, err error) error {
		if d != nil && d.IsDir() {
			return nil
		}
		count++
		assert.Contains(t, templateFiles, path)
		return nil
	}))
	assert.Equal(t, len(templateFiles), count, "Must match the expected number of calls")

}
