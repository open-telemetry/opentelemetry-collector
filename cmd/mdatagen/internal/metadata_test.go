// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"io/fs"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		wantErr string
	}{
		{
			name:    "testdata/no_type.yaml",
			wantErr: "missing type",
		},
		{
			name:    "testdata/no_status.yaml",
			wantErr: "missing status",
		},
		{
			name:    "testdata/no_class.yaml",
			wantErr: "missing class",
		},
		{
			name:    "testdata/invalid_class.yaml",
			wantErr: "invalid class: incorrectclass",
		},
		{
			name:    "testdata/no_stability.yaml",
			wantErr: "missing stability",
		},
		{
			name:    "testdata/no_deprecation_info.yaml",
			wantErr: "deprecated component missing deprecation date and migration guide for traces",
		},
		{
			name:    "testdata/no_deprecation_date_info.yaml",
			wantErr: "deprecated component missing date in YYYY-MM-DD format: traces",
		},
		{
			name:    "testdata/no_deprecation_migration_info.yaml",
			wantErr: "deprecated component missing migration guide: traces",
		},
		{
			name:    "testdata/deprecation_info_invalid_date.yaml",
			wantErr: "deprecated component missing valid date in YYYY-MM-DD format: traces",
		},
		{
			name:    "testdata/invalid_stability.yaml",
			wantErr: "decoding failed due to the following error(s):\n\n'status.stability' unsupported stability level: \"incorrectstability\"",
		},
		{
			name:    "testdata/no_stability_component.yaml",
			wantErr: "missing component for stability: Beta",
		},
		{
			name:    "testdata/invalid_stability_component.yaml",
			wantErr: "invalid component: incorrectcomponent",
		},
		{
			name:    "testdata/no_description_rattr.yaml",
			wantErr: "empty description for resource attribute: string.resource.attr",
		},
		{
			name:    "testdata/no_type_rattr.yaml",
			wantErr: "empty type for resource attribute: string.resource.attr",
		},
		{
			name:    "testdata/no_metric_description.yaml",
			wantErr: "metric \"default.metric\": missing metric description",
		},
		{
			name:    "testdata/events/no_description.yaml",
			wantErr: "event \"default.event\": missing event description",
		},
		{
			name:    "testdata/no_metric_unit.yaml",
			wantErr: "metric \"default.metric\": missing metric unit",
		},
		{
			name: "testdata/no_metric_type.yaml",
			wantErr: "metric \"system.cpu.time\": missing metric type key, " +
				"one of the following has to be specified: sum, gauge, histogram",
		},
		{
			name: "testdata/two_metric_types.yaml",
			wantErr: "metric \"system.cpu.time\": more than one metric type keys, " +
				"only one of the following has to be specified: sum, gauge, histogram",
		},
		{
			name:    "testdata/invalid_input_type.yaml",
			wantErr: "metric \"system.cpu.time\": invalid `input_type` value \"double\", must be \"\" or \"string\"",
		},
		{
			name:    "testdata/unknown_metric_attribute.yaml",
			wantErr: "metric \"system.cpu.time\" refers to undefined attributes: [missing]",
		},
		{
			name:    "testdata/events/unknown_attribute.yaml",
			wantErr: "event \"system.event\" refers to undefined attributes: [missing]",
		},
		{
			name:    "testdata/unused_attribute.yaml",
			wantErr: "unused attributes: [unused_attr]",
		},
		{
			name:    "testdata/no_description_attr.yaml",
			wantErr: "missing attribute description for: string_attr",
		},
		{
			name:    "testdata/no_type_attr.yaml",
			wantErr: "empty type for attribute: used_attr",
		},
		{
			name:    "testdata/entity_undefined_id_attribute.yaml",
			wantErr: `entity "host": identity refers to undefined resource attribute: host.missing`,
		},
		{
			name:    "testdata/entity_undefined_description_attribute.yaml",
			wantErr: `entity "host": description refers to undefined resource attribute: host.missing`,
		},
		{
			name:    "testdata/entity_empty_id_attributes.yaml",
			wantErr: `entity "host": identity is required`,
		},
		{
			name:    "testdata/entity_duplicate_attributes.yaml",
			wantErr: `attribute host.name is already used by entity`,
		},
		{
			name:    "testdata/entity_duplicate_types.yaml",
			wantErr: `duplicate entity type: host`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadMetadata(tt.name)
			require.Error(t, err)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestValidateMetricDuplicates(t *testing.T) {
	allowedMetrics := map[string][]string{
		"container.cpu.utilization": {"docker_stats", "kubeletstats"},
		"container.memory.rss":      {"docker_stats", "kubeletstats"},
		"container.uptime":          {"docker_stats", "kubeletstats"},
	}
	allMetrics := map[string][]string{}
	err := filepath.Walk("../../../receiver", func(path string, info fs.FileInfo, _ error) error {
		if info.Name() == "metadata.yaml" {
			md, err := LoadMetadata(path)
			require.NoError(t, err)
			if len(md.Metrics) > 0 {
				for metricName := range md.Metrics {
					allMetrics[md.Type] = append(allMetrics[md.Type], string(metricName))
				}
			}
		}
		return nil
	})
	require.NoError(t, err)

	seen := make(map[string]string)
	for receiver, metrics := range allMetrics {
		for _, metricName := range metrics {
			if val, exists := seen[metricName]; exists {
				receivers, allowed := allowedMetrics[metricName]
				assert.Truef(
					t,
					allowed && slices.Contains(receivers, receiver) && slices.Contains(receivers, val),
					"Duplicate metric %v in receivers %v and %v. Please validate that this is intentional by adding the metric name and receiver types in the allowedMetrics map in this test\n", metricName, receiver, val,
				)
			}
			seen[metricName] = receiver
		}
	}
}

func TestSupportsSignal(t *testing.T) {
	md := Metadata{}
	assert.False(t, md.supportsSignal("logs"))
}

func TestCodeCovID(t *testing.T) {
	tests := []struct {
		md   Metadata
		want string
	}{
		{
			md: Metadata{
				Type: "aes",
				Status: &Status{
					Class:              "provider",
					CodeCovComponentID: "my_custom_id",
				},
			},
			want: "my_custom_id",
		},
		{
			md: Metadata{
				Type: "count",
				Status: &Status{
					Class: "connector",
				},
			},
			want: "connector_count",
		},
		{
			md: Metadata{
				Type: "file",
				Status: &Status{
					Class: "exporter",
				},
			},
			want: "exporter_file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.md.Type, func(t *testing.T) {
			got := tt.md.GetCodeCovComponentID()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAttributeRequirementLevel(t *testing.T) {
	tests := []struct {
		name             string
		requirementLevel AttributeRequirementLevel
		wantConditional  bool
	}{
		{
			name:             "required",
			requirementLevel: AttributeRequirementLevelRequired,
			wantConditional:  false,
		},
		{
			name:             "conditionally_required",
			requirementLevel: AttributeRequirementLevelConditionallyRequired,
			wantConditional:  true,
		},
		{
			name:             "recommended",
			requirementLevel: AttributeRequirementLevelRecommended,
			wantConditional:  false,
		},
		{
			name:             "opt_in",
			requirementLevel: AttributeRequirementLevelOptIn,
			wantConditional:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := Attribute{RequirementLevel: tt.requirementLevel}
			assert.Equal(t, tt.wantConditional, attr.IsConditional())
		})
	}
}

func TestAttributeRequirementLevelUnmarshalText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    AttributeRequirementLevel
		wantErr bool
	}{
		{
			name:  "required",
			input: "required",
			want:  AttributeRequirementLevelRequired,
		},
		{
			name:  "conditionally_required",
			input: "conditionally_required",
			want:  AttributeRequirementLevelConditionallyRequired,
		},
		{
			name:  "recommended",
			input: "recommended",
			want:  AttributeRequirementLevelRecommended,
		},
		{
			name:  "opt_in",
			input: "opt_in",
			want:  AttributeRequirementLevelOptIn,
		},
		{
			name:  "empty defaults to recommended",
			input: "",
			want:  AttributeRequirementLevelRecommended,
		},
		{
			name:    "invalid value",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rl AttributeRequirementLevel
			err := rl.UnmarshalText([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, rl)
		})
	}
}
