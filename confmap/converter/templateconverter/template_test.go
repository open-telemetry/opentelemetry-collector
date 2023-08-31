// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package templateconverter

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestTemplateConverter(t *testing.T) {
	var testCases = []struct {
		name      string // test case name (also directory name containing config files)
		templates []string
	}{
		{
			name:      "single-receiver",
			templates: []string{"my_filelog_template"},
		},
		{
			name:      "multiple-receivers",
			templates: []string{"my_filelog_template"},
		},
		{
			name:      "single-processor",
			templates: []string{"my_filelog_template"},
		},
		{
			name:      "multiple-processors",
			templates: []string{"my_filelog_template"},
		},
		{
			name:      "multiple-pipelines",
			templates: []string{"my_filelog_template"},
		},
		{
			name:      "multiple-templates",
			templates: []string{"my_filelog_template", "my_otlp_template"},
		},
		{
			name:      "multiple-template-instances",
			templates: []string{"my_filelog_template"},
		},
		{
			name:      "auto-skip-traces",
			templates: []string{"my_otlp_template"},
		},
		{
			name:      "auto-skip-metrics",
			templates: []string{"my_otlp_template"},
		},
		{
			name:      "auto-skip-logs",
			templates: []string{"my_otlp_template"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conf, err := confmaptest.LoadConf(filepath.Join("testdata", tc.name, "config.yaml"))
			require.NoError(t, err)

			expected, err := confmaptest.LoadConf(filepath.Join("testdata", tc.name, "expected.yaml"))
			require.NoError(t, err)

			templates := make(map[string]any, len(tc.templates))
			for _, template := range tc.templates {
				// A "real" template file should have a top level structure that includes
				// 'type' and 'template' keys, but that is a concern of the template provider.
				// For the sake of testing the converter, we'll assume the file name is the type
				// and the contents are the template.
				templateFilename := filepath.Join("testdata", tc.name, template+".yaml")
				templateContent, err := os.ReadFile(templateFilename) // nolint: gosec
				require.NoError(t, err)

				templates[template] = string(templateContent)
			}

			require.NoError(t, conf.Merge(confmap.NewFromStringMap(map[string]any{"templates": templates})))

			require.NoError(t, New().Convert(context.Background(), conf))
			expectedMap := expected.ToStringMap()
			actualMap := conf.ToStringMap()
			assert.Equal(t, expectedMap, actualMap)
		})
	}
}
