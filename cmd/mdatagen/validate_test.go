// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
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
			name:    "testdata/invalid_stability.yaml",
			wantErr: "invalid stability: incorrectstability",
		},
		{
			name:    "testdata/no_stability_component.yaml",
			wantErr: "missing component for stability: beta",
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
			name:    "testdata/no_metric_unit.yaml",
			wantErr: "metric \"default.metric\": missing metric unit",
		},
		{
			name: "testdata/no_metric_type.yaml",
			wantErr: "metric system.cpu.time doesn't have a metric type key, " +
				"one of the following has to be specified: sum, gauge",
		},
		{
			name: "testdata/two_metric_types.yaml",
			wantErr: "metric system.cpu.time has more than one metric type keys, " +
				"only one of the following has to be specified: sum, gauge",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadMetadata(tt.name)
			require.Error(t, err)
			require.EqualError(t, err, tt.wantErr)
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
	err := filepath.Walk("../../receiver", func(path string, info fs.FileInfo, err error) error {
		if info.Name() == "metadata.yaml" {
			md, err := loadMetadata(path)
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
				assert.True(
					t,
					allowed && contains(receiver, receivers) && contains(val, receivers),
					fmt.Sprintf("Duplicate metric %v in receivers %v and %v. Please validate that this is intentional by adding the metric name and receiver types in the allowedMetrics map in this test\n", metricName, receiver, val),
				)
			}
			seen[metricName] = receiver
		}
	}
}

func contains(r string, rs []string) bool {
	for _, s := range rs {
		if s == r {
			return true
		}
	}
	return false
}
