// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package templateconverter

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestTemplateConverter(t *testing.T) {
	for _, tc := range []string{
		"no-templates",
		"single-receiver-no-pipelines",
		"single-receiver",
		"single-processor",
		"multiple-receivers-no-pipelines",
		"multiple-receivers-single-pipeline",
		"multiple-receivers-separate-pipelines",
		"multiple-processors",
		"multiple-pipelines",
		"multiple-templates",
		"multiple-template-instances",
		"auto-skip-traces",
		"auto-skip-metrics",
		"auto-skip-logs",
		"extra-pipeline",
		"extra-receiver",
		"only-processor-templated",
		"extra-receiver-no-pipelines",
		"unused-template",
	} {
		t.Run(tc, func(t *testing.T) {
			conf, err := confmaptest.LoadConf(filepath.Join("testdata", "valid", tc, "config.yaml"))
			require.NoError(t, err)

			expected, err := confmaptest.LoadConf(filepath.Join("testdata", "valid", tc, "expected.yaml"))
			require.NoError(t, err)

			require.NoError(t, New().Convert(context.Background(), conf))
			expectedMap := expected.ToStringMap()
			actualMap := conf.ToStringMap()
			assert.Equal(t, expectedMap, actualMap)
		})
	}
}

func TestRenderError(t *testing.T) {
	var testCases = []struct {
		name      string
		expectErr string
	}{
		{
			name:      "template-not-found",
			expectErr: "template type \"non_existent\" not found",
		},
		{
			name:      "malformed-template-id",
			expectErr: "'template' must be followed by type",
		},
		{
			name:      "malformed-template-structure",
			expectErr: "malformed: yaml: unmarshal errors",
		},
		{
			name:      "malformed-template-syntax",
			expectErr: "template: my_filelog_template:4: unexpected",
		},
		{
			name:      "missing-receiver",
			expectErr: "must have at least one receiver",
		},
		{
			name:      "missing-pipeline",
			expectErr: "template containing processors must have at least one pipeline",
		},
		{
			name:      "templates-wrong-type",
			expectErr: "'templates' must be a map",
		},
		{
			name:      "templates-receivers-missing",
			expectErr: "'templates' must contain a 'receivers' section",
		},
		{
			name:      "templates-receivers-wrong-type",
			expectErr: "'templates::receivers' must be a map",
		},
		{
			name:      "templates-receivers-raw-wrong-type",
			expectErr: "'templates::receivers::my_filelog_template' must be a string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conf, err := confmaptest.LoadConf(filepath.Join("testdata", "invalid", tc.name+".yaml"))
			require.NoError(t, err)

			err = New().Convert(context.Background(), conf)
			// validate that err is not nil and that it contains the expected error message
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectErr)
		})
	}
}
