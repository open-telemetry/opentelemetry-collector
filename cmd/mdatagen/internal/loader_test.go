// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestTwoPackagesInDirectory(t *testing.T) {
	contents, err := os.ReadFile("testdata/twopackages.yaml")
	require.NoError(t, err)
	tempDir := t.TempDir()
	metadataPath := filepath.Join(tempDir, "metadata.yaml")
	// we create a trivial module and packages to avoid having invalid go checked into our test directory.
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module twopackages"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "package1.go"), []byte("package package1"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "package2.go"), []byte("package package2"), 0o600))
	require.NoError(t, os.WriteFile(metadataPath, contents, 0o600))

	_, err = LoadMetadata(metadataPath)
	require.Error(t, err)
	require.ErrorContains(t, err, "unable to determine package name: [go list -f {{.ImportPath}}] failed: (stderr) found packages package1 (package1.go) and package2 (package2.go)")
}

func TestLoadMetadata(t *testing.T) {
	tests := []struct {
		name    string
		want    Metadata
		wantErr string
	}{
		{
			name: "samplereceiver/metadata.yaml",
			want: Metadata{
				GithubProject:        "open-telemetry/opentelemetry-collector",
				GeneratedPackageName: "metadata",
				Type:                 "sample",
				SemConvVersion:       "1.9.0",
				Status: &Status{
					DisableCodeCov: true,
					Class:          "receiver",
					Stability: map[component.StabilityLevel][]string{
						component.StabilityLevelDevelopment: {"logs"},
						component.StabilityLevelBeta:        {"traces"},
						component.StabilityLevelStable:      {"metrics"},
						component.StabilityLevelDeprecated:  {"profiles"},
					},
					Distributions: []string{},
					Deprecation: DeprecationMap{
						"profiles": DeprecationInfo{
							Date:      "2025-02-05",
							Migration: "no migration needed",
						},
					},
					Codeowners: &Codeowners{
						Active: []string{"dmitryax"},
					},
					Warnings:             []string{"Any additional information that should be brought to the consumer's attention"},
					UnsupportedPlatforms: []string{"freebsd", "illumos"},
				},
				ResourceAttributes: map[AttributeName]Attribute{
					"string.resource.attr": {
						Description: "Resource attribute with any string value.",
						Enabled:     true,
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName: "string.resource.attr",
					},
					"string.enum.resource.attr": {
						Description: "Resource attribute with a known set of string values.",
						Enabled:     true,
						Enum:        []string{"one", "two"},
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName: "string.enum.resource.attr",
					},
					"optional.resource.attr": {
						Description: "Explicitly disabled ResourceAttribute.",
						Enabled:     false,
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName: "optional.resource.attr",
					},
					"slice.resource.attr": {
						Description: "Resource attribute with a slice value.",
						Enabled:     true,
						Type: ValueType{
							ValueType: pcommon.ValueTypeSlice,
						},
						FullName: "slice.resource.attr",
					},
					"map.resource.attr": {
						Description: "Resource attribute with a map value.",
						Enabled:     true,
						Type: ValueType{
							ValueType: pcommon.ValueTypeMap,
						},
						FullName: "map.resource.attr",
					},
					"string.resource.attr_disable_warning": {
						Description: "Resource attribute with any string value.",
						Warnings: Warnings{
							IfEnabledNotSet: "This resource_attribute will be disabled by default soon.",
						},
						Enabled: true,
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName: "string.resource.attr_disable_warning",
					},
					"string.resource.attr_remove_warning": {
						Description: "Resource attribute with any string value.",
						Warnings: Warnings{
							IfConfigured: "This resource_attribute is deprecated and will be removed soon.",
						},
						Enabled: false,
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName: "string.resource.attr_remove_warning",
					},
					"string.resource.attr_to_be_removed": {
						Description: "Resource attribute with any string value.",
						Warnings: Warnings{
							IfEnabled: "This resource_attribute is deprecated and will be removed soon.",
						},
						Enabled: true,
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName: "string.resource.attr_to_be_removed",
					},
				},

				Attributes: map[AttributeName]Attribute{
					"enum_attr": {
						Description:  "Attribute with a known set of string values.",
						NameOverride: "",
						Enum:         []string{"red", "green", "blue"},
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName: "enum_attr",
					},
					"string_attr": {
						Description:  "Attribute with any string value.",
						NameOverride: "",
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName: "string_attr",
					},
					"overridden_int_attr": {
						Description:  "Integer attribute with overridden name.",
						NameOverride: "state",
						Type: ValueType{
							ValueType: pcommon.ValueTypeInt,
						},
						FullName: "overridden_int_attr",
					},
					"boolean_attr": {
						Description: "Attribute with a boolean value.",
						Type: ValueType{
							ValueType: pcommon.ValueTypeBool,
						},
						FullName: "boolean_attr",
					},
					"boolean_attr2": {
						Description: "Another attribute with a boolean value.",
						Type: ValueType{
							ValueType: pcommon.ValueTypeBool,
						},
						FullName: "boolean_attr2",
					},
					"slice_attr": {
						Description: "Attribute with a slice value.",
						Type: ValueType{
							ValueType: pcommon.ValueTypeSlice,
						},
						FullName: "slice_attr",
					},
					"map_attr": {
						Description: "Attribute with a map value.",
						Type: ValueType{
							ValueType: pcommon.ValueTypeMap,
						},
						FullName: "map_attr",
					},
					"optional_int_attr": {
						Description: "An optional attribute with an integer value",
						Type: ValueType{
							ValueType: pcommon.ValueTypeInt,
						},
						FullName: "optional_int_attr",
						Optional: true,
					},
					"optional_string_attr": {
						Description: "An optional attribute with any string value",
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName: "optional_string_attr",
						Optional: true,
					},
				},
				Metrics: map[MetricName]Metric{
					"default.metric": {
						Signal: Signal{
							Enabled:               true,
							Description:           "Monotonic cumulative sum int metric enabled by default.",
							ExtendedDocumentation: "The metric will be become optional soon.",
							Warnings: Warnings{
								IfEnabledNotSet: "This metric will be disabled by default soon.",
							},
							Attributes: []AttributeName{"string_attr", "overridden_int_attr", "enum_attr", "slice_attr", "map_attr", "optional_int_attr", "optional_string_attr"},
						},
						Unit: strPtr("s"),
						Sum: &Sum{
							MetricValueType:        MetricValueType{pmetric.NumberDataPointValueTypeInt},
							AggregationTemporality: AggregationTemporality{Aggregation: pmetric.AggregationTemporalityCumulative},
							Mono:                   Mono{Monotonic: true},
						},
					},
					"optional.metric": {
						Signal: Signal{
							Enabled:     false,
							Description: "[DEPRECATED] Gauge double metric disabled by default.",
							Warnings: Warnings{
								IfConfigured: "This metric is deprecated and will be removed soon.",
							},
							Attributes: []AttributeName{"string_attr", "boolean_attr", "boolean_attr2", "optional_string_attr"},
						},
						Unit: strPtr("1"),
						Gauge: &Gauge{
							MetricValueType: MetricValueType{pmetric.NumberDataPointValueTypeDouble},
						},
					},
					"optional.metric.empty_unit": {
						Signal: Signal{
							Enabled:     false,
							Description: "[DEPRECATED] Gauge double metric disabled by default.",
							Warnings: Warnings{
								IfConfigured: "This metric is deprecated and will be removed soon.",
							},
							Attributes: []AttributeName{"string_attr", "boolean_attr"},
						},
						Unit: strPtr(""),
						Gauge: &Gauge{
							MetricValueType: MetricValueType{pmetric.NumberDataPointValueTypeDouble},
						},
					},

					"default.metric.to_be_removed": {
						Signal: Signal{
							Enabled:               true,
							Description:           "[DEPRECATED] Non-monotonic delta sum double metric enabled by default.",
							ExtendedDocumentation: "The metric will be removed soon.",
							Warnings: Warnings{
								IfEnabled: "This metric is deprecated and will be removed soon.",
							},
						},
						Unit: strPtr("s"),
						Sum: &Sum{
							MetricValueType:        MetricValueType{pmetric.NumberDataPointValueTypeDouble},
							AggregationTemporality: AggregationTemporality{Aggregation: pmetric.AggregationTemporalityDelta},
							Mono:                   Mono{Monotonic: false},
						},
					},
					"metric.input_type": {
						Signal: Signal{
							Enabled:     true,
							Description: "Monotonic cumulative sum int metric with string input_type enabled by default.",
							Attributes:  []AttributeName{"string_attr", "overridden_int_attr", "enum_attr", "slice_attr", "map_attr"},
						},
						Unit: strPtr("s"),
						Sum: &Sum{
							MetricValueType:        MetricValueType{pmetric.NumberDataPointValueTypeInt},
							MetricInputType:        MetricInputType{InputType: "string"},
							AggregationTemporality: AggregationTemporality{Aggregation: pmetric.AggregationTemporalityCumulative},
							Mono:                   Mono{Monotonic: true},
						},
					},
				},
				Events: map[EventName]Event{
					"default.event": {
						Signal: Signal{
							Enabled:     true,
							Description: "Example event enabled by default.",
							Warnings: Warnings{
								IfEnabledNotSet: "This event will be disabled by default soon.",
							},
							Attributes: []AttributeName{"string_attr", "overridden_int_attr", "enum_attr", "slice_attr", "map_attr", "optional_int_attr", "optional_string_attr"},
						},
					},
					"default.event.to_be_renamed": {
						Signal: Signal{
							Enabled:               false,
							Description:           "[DEPRECATED] Example event disabled by default.",
							ExtendedDocumentation: "The event will be renamed soon.",
							Warnings: Warnings{
								IfConfigured: "This event is deprecated and will be renamed soon.",
							},
							Attributes: []AttributeName{"string_attr", "boolean_attr", "boolean_attr2", "optional_string_attr"},
						},
					},
					"default.event.to_be_removed": {
						Signal: Signal{
							Enabled:               true,
							Description:           "[DEPRECATED] Example to-be-removed event enabled by default.",
							ExtendedDocumentation: "The event will be removed soon.",
							Warnings: Warnings{
								IfEnabled: "This event is deprecated and will be removed soon.",
							},
							Attributes: []AttributeName{"string_attr", "overridden_int_attr", "enum_attr", "slice_attr", "map_attr"},
						},
					},
				},
				Telemetry: Telemetry{
					Metrics: map[MetricName]Metric{
						"batch_size_trigger_send": {
							Signal: Signal{
								Enabled:     true,
								Stability:   Stability{Level: "deprecated", From: "v0.110.0"},
								Description: "Number of times the batch was sent due to a size trigger",
							},
							Unit: strPtr("{times}"),
							Sum: &Sum{
								MetricValueType: MetricValueType{pmetric.NumberDataPointValueTypeInt},
								Mono:            Mono{Monotonic: true},
							},
						},
						"request_duration": {
							Signal: Signal{
								Enabled:     true,
								Stability:   Stability{Level: "alpha"},
								Description: "Duration of request",
							},
							Unit: strPtr("s"),
							Histogram: &Histogram{
								MetricValueType: MetricValueType{pmetric.NumberDataPointValueTypeDouble},
								Boundaries:      []float64{1, 10, 100},
							},
						},
						"process_runtime_total_alloc_bytes": {
							Signal: Signal{
								Enabled:     true,
								Stability:   Stability{Level: "stable"},
								Description: "Cumulative bytes allocated for heap objects (see 'go doc runtime.MemStats.TotalAlloc')",
							},
							Unit: strPtr("By"),
							Sum: &Sum{
								Mono: Mono{true},
								MetricValueType: MetricValueType{
									ValueType: pmetric.NumberDataPointValueTypeInt,
								},
								Async: true,
							},
						},
						"queue_length": {
							Signal: Signal{
								Enabled:               true,
								Stability:             Stability{Level: "alpha"},
								Description:           "This metric is optional and therefore not initialized in NewTelemetryBuilder.",
								ExtendedDocumentation: "For example this metric only exists if feature A is enabled.",
							},
							Unit:     strPtr("{items}"),
							Optional: true,
							Gauge: &Gauge{
								MetricValueType: MetricValueType{
									ValueType: pmetric.NumberDataPointValueTypeInt,
								},
								Async: true,
							},
						},
						"queue_capacity": {
							Signal: Signal{
								Enabled:     true,
								Description: "Queue capacity - sync gauge example.",
							},
							Unit: strPtr("{items}"),
							Gauge: &Gauge{
								MetricValueType: MetricValueType{
									ValueType: pmetric.NumberDataPointValueTypeInt,
								},
							},
						},
					},
				},
				ScopeName:       "go.opentelemetry.io/collector/internal/receiver/samplereceiver",
				ShortFolderName: "sample",
				Tests:           Tests{Host: "componenttest.NewNopHost()"},
			},
		},
		{
			name: "testdata/parent.yaml",
			want: Metadata{
				Type:                 "subcomponent",
				Parent:               "parentComponent",
				GeneratedPackageName: "metadata",
				ScopeName:            "go.opentelemetry.io/collector/cmd/mdatagen/internal/testdata",
				ShortFolderName:      "testdata",
				Tests:                Tests{Host: "componenttest.NewNopHost()"},
			},
		},
		{
			name: "testdata/generated_package_name.yaml",
			want: Metadata{
				Type:                 "custom",
				GeneratedPackageName: "customname",
				ScopeName:            "go.opentelemetry.io/collector/cmd/mdatagen/internal/testdata",
				ShortFolderName:      "testdata",
				Tests:                Tests{Host: "componenttest.NewNopHost()"},
				Status: &Status{
					Class: "receiver",
					Stability: map[component.StabilityLevel][]string{
						component.StabilityLevelDevelopment: {"logs"},
						component.StabilityLevelBeta:        {"traces"},
						component.StabilityLevelStable:      {"metrics"},
					},
				},
			},
		},
		{
			name: "testdata/empty_test_config.yaml",
			want: Metadata{
				Type:                 "test",
				GeneratedPackageName: "metadata",
				ScopeName:            "go.opentelemetry.io/collector/cmd/mdatagen/internal/testdata",
				ShortFolderName:      "testdata",
				Tests:                Tests{Host: "componenttest.NewNopHost()"},
				Status: &Status{
					Class: "receiver",
					Stability: map[component.StabilityLevel][]string{
						component.StabilityLevelBeta: {"logs"},
					},
				},
			},
		},
		{
			name:    "testdata/invalid_type_rattr.yaml",
			want:    Metadata{},
			wantErr: "decoding failed due to the following error(s):\n\n'resource_attributes[string.resource.attr].type' invalid type: \"invalidtype\"",
		},
		{
			name:    "testdata/no_enabled.yaml",
			want:    Metadata{},
			wantErr: "decoding failed due to the following error(s):\n\n'metrics[system.cpu.time]' missing required field: `enabled`",
		},
		{
			name:    "testdata/events/no_enabled.yaml",
			want:    Metadata{},
			wantErr: "decoding failed due to the following error(s):\n\n'events[system.event]' missing required field: `enabled`",
		},
		{
			name: "testdata/no_value_type.yaml",
			want: Metadata{},
			wantErr: "decoding failed due to the following error(s):\n\n'metrics[system.cpu.time]' decoding failed due to the following error(s):\n\n" +
				"'sum' missing required field: `value_type`",
		},
		{
			name:    "testdata/unknown_value_type.yaml",
			wantErr: "decoding failed due to the following error(s):\n\n'metrics[system.cpu.time]' decoding failed due to the following error(s):\n\n'sum' decoding failed due to the following error(s):\n\n'value_type' invalid value_type: \"unknown\"",
		},
		{
			name:    "testdata/invalid_aggregation.yaml",
			want:    Metadata{},
			wantErr: "decoding failed due to the following error(s):\n\n'metrics[default.metric]' decoding failed due to the following error(s):\n\n'sum' decoding failed due to the following error(s):\n\n'aggregation_temporality' invalid aggregation: \"invalidaggregation\"",
		},
		{
			name:    "testdata/invalid_type_attr.yaml",
			want:    Metadata{},
			wantErr: "decoding failed due to the following error(s):\n\n'attributes[used_attr].type' invalid type: \"invalidtype\"",
		},
		{
			name:    "testdata/~~this file doesn't exist~~.yaml",
			wantErr: "unable to read the file file:testdata/~~this file doesn't exist~~.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadMetadata(tt.name)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
