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
			name:      "single-receiver-no-pipelines",
			templates: []string{"my_filelog_template"},
		},
		{
			name:      "single-receiver",
			templates: []string{"my_filelog_template"},
		},
		{
			name:      "single-processor",
			templates: []string{"my_filelog_template"},
		},
		{
			name:      "multiple-receivers-no-pipelines",
			templates: []string{"my_filelog_template"},
		},
		{
			name:      "multiple-receivers-single-pipeline",
			templates: []string{"my_filelog_template"},
		},
		{
			name:      "multiple-receivers-separate-pipelines",
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
		{
			name:      "extra-pipeline",
			templates: []string{"my_filelog_template"},
		},
		{
			name:      "extra-receiver",
			templates: []string{"my_filelog_template"},
		},
		{
			name:      "only-processor-templated",
			templates: []string{"my_filelog_template"},
		},
		{
			name:      "extra-receiver-no-pipelines",
			templates: []string{"my_filelog_template"},
		},
		{
			name:      "unused-template",
			templates: []string{"my_filelog_template", "my_otlp_template"},
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
				templates[template] = readTemplate(t, filepath.Join("testdata", tc.name, template+".yaml"))
			}

			require.NoError(t, conf.Merge(confmap.NewFromStringMap(map[string]any{"templates": templates})))

			require.NoError(t, New().Convert(context.Background(), conf))
			expectedMap := expected.ToStringMap()
			actualMap := conf.ToStringMap()
			assert.Equal(t, expectedMap, actualMap)
		})
	}
}

func TestNoTemplates(t *testing.T) {
	rawConf := map[string]any{
		"receivers": map[string]any{
			"my_receiver": map[string]any{
				"foo": "bar",
			},
		},
		"exporters": map[string]any{
			"my_exporter": map[string]any{
				"foo": "bar",
			},
		},
		"service": map[string]any{
			"pipelines": map[string]any{
				"my_pipeline": map[string]any{
					"receivers": []string{"my_receiver"},
					"exporters": []string{"my_exporter"},
				},
			},
		},
	}

	converted := confmap.NewFromStringMap(rawConf)
	require.NoError(t, New().Convert(context.Background(), converted))
	assert.Equal(t, rawConf, converted.ToStringMap())
}

func TestRenderError(t *testing.T) {
	var testCases = []struct {
		name      string // test case name (also directory name containing config files)
		templates []string
		expectErr string
	}{
		{
			name:      "template-not-found",
			templates: []string{"my_filelog_template"},
			expectErr: "template type \"non_existent\" not found",
		},
		{
			name:      "malformed-template-id",
			templates: []string{"my_filelog_template"},
			expectErr: "'template' must be followed by type",
		},
		{
			name:      "malformed-template",
			templates: []string{"my_filelog_template"},
			expectErr: "malformed: yaml: unmarshal errors",
		},
		{
			name:      "missing-receiver",
			templates: []string{"my_filelog_template"},
			expectErr: "must have at least one receiver",
		},
		{
			name:      "missing-pipeline",
			templates: []string{"my_filelog_template"},
			expectErr: "template containing processors must have at least one pipeline",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conf, err := confmaptest.LoadConf(filepath.Join("testdata", tc.name, "config.yaml"))
			require.NoError(t, err)

			templates := make(map[string]any, len(tc.templates))
			for _, template := range tc.templates {
				// A "real" template file should have a top level structure that includes
				// 'type' and 'template' keys, but that is a concern of the template provider.
				// For the sake of testing the converter, we'll assume the file name is the type
				// and the contents are the template.
				templates[template] = readTemplate(t, filepath.Join("testdata", tc.name, template+".yaml"))
			}

			require.NoError(t, conf.Merge(confmap.NewFromStringMap(map[string]any{"templates": templates})))

			err = New().Convert(context.Background(), conf)
			// validate that err is not nil and that it contains the expected error message
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectErr)
		})
	}
}

func readTemplate(t *testing.T, filename string) string {
	templateContent, err := os.ReadFile(filename) // nolint: gosec
	require.NoError(t, err)
	return string(templateContent)
}
