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

func boolPtr(b bool) *bool {
	return &b
}

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
				DisplayName:          "Sample Receiver",
				Description:          "This receiver is used for testing purposes to check the output of mdatagen.",
				SemConvVersion:       "1.38.0",
				PackageName:          "go.opentelemetry.io/collector/cmd/mdatagen/internal/samplereceiver",
				ReaggregationEnabled: true,
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
						EnabledPtr:  boolPtr(true),
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName:         "string.resource.attr",
						RequirementLevel: AttributeRequirementLevelRecommended,
					},
					"string.enum.resource.attr": {
						Description: "Resource attribute with a known set of string values.",
						EnabledPtr:  boolPtr(true),
						Enum:        []string{"one", "two"},
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName:         "string.enum.resource.attr",
						RequirementLevel: AttributeRequirementLevelRecommended,
					},
					"optional.resource.attr": {
						Description: "Explicitly disabled ResourceAttribute.",
						EnabledPtr:  boolPtr(false),
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName:         "optional.resource.attr",
						RequirementLevel: AttributeRequirementLevelRecommended,
					},
					"slice.resource.attr": {
						Description: "Resource attribute with a slice value.",
						EnabledPtr:  boolPtr(true),
						Type: ValueType{
							ValueType: pcommon.ValueTypeSlice,
						},
						FullName:         "slice.resource.attr",
						RequirementLevel: AttributeRequirementLevelRecommended,
					},
					"map.resource.attr": {
						Description: "Resource attribute with a map value.",
						EnabledPtr:  boolPtr(true),
						Type: ValueType{
							ValueType: pcommon.ValueTypeMap,
						},
						FullName:         "map.resource.attr",
						RequirementLevel: AttributeRequirementLevelRecommended,
					},
					"string.resource.attr_disable_warning": {
						Description: "Resource attribute with any string value.",
						Warnings: Warnings{
							IfEnabledNotSet: "This resource_attribute will be disabled by default soon.",
						},
						EnabledPtr: boolPtr(true),
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName:         "string.resource.attr_disable_warning",
						RequirementLevel: AttributeRequirementLevelRecommended,
					},
					"string.resource.attr_remove_warning": {
						Description: "Resource attribute with any string value.",
						Warnings: Warnings{
							IfConfigured: "This resource_attribute is deprecated and will be removed soon.",
						},
						EnabledPtr: boolPtr(false),
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName:         "string.resource.attr_remove_warning",
						RequirementLevel: AttributeRequirementLevelRecommended,
					},
					"string.resource.attr_to_be_removed": {
						Description: "Resource attribute with any string value.",
						Warnings: Warnings{
							IfEnabled: "This resource_attribute is deprecated and will be removed soon.",
						},
						EnabledPtr: boolPtr(true),
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName:         "string.resource.attr_to_be_removed",
						RequirementLevel: AttributeRequirementLevelRecommended,
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
						FullName:         "enum_attr",
						RequirementLevel: AttributeRequirementLevelRecommended,
					},
					"string_attr": {
						Description:  "Attribute with any string value.",
						NameOverride: "",
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName:         "string_attr",
						RequirementLevel: AttributeRequirementLevelRecommended,
					},
					"overridden_int_attr": {
						Description:  "Integer attribute with overridden name.",
						NameOverride: "state",
						Type: ValueType{
							ValueType: pcommon.ValueTypeInt,
						},
						FullName:         "overridden_int_attr",
						RequirementLevel: AttributeRequirementLevelRecommended,
					},
					"boolean_attr": {
						Description: "Attribute with a boolean value.",
						Type: ValueType{
							ValueType: pcommon.ValueTypeBool,
						},
						FullName:         "boolean_attr",
						RequirementLevel: AttributeRequirementLevelRecommended,
					},
					"boolean_attr2": {
						Description: "Another attribute with a boolean value.",
						Type: ValueType{
							ValueType: pcommon.ValueTypeBool,
						},
						FullName:         "boolean_attr2",
						RequirementLevel: AttributeRequirementLevelRecommended,
					},
					"slice_attr": {
						Description: "Attribute with a slice value.",
						Type: ValueType{
							ValueType: pcommon.ValueTypeSlice,
						},
						FullName:         "slice_attr",
						RequirementLevel: AttributeRequirementLevelRecommended,
					},
					"map_attr": {
						Description: "Attribute with a map value.",
						Type: ValueType{
							ValueType: pcommon.ValueTypeMap,
						},
						FullName:         "map_attr",
						RequirementLevel: AttributeRequirementLevelRecommended,
					},
					"conditional_int_attr": {
						Description: "A conditional attribute with an integer value",
						Type: ValueType{
							ValueType: pcommon.ValueTypeInt,
						},
						FullName:         "conditional_int_attr",
						RequirementLevel: AttributeRequirementLevelConditionallyRequired,
					},
					"conditional_string_attr": {
						Description: "A conditional attribute with any string value",
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName:         "conditional_string_attr",
						RequirementLevel: AttributeRequirementLevelConditionallyRequired,
					},
					"opt_in_bool_attr": {
						Description: "An opt-in attribute with a boolean value",
						Type: ValueType{
							ValueType: pcommon.ValueTypeBool,
						},
						FullName:         "opt_in_bool_attr",
						RequirementLevel: AttributeRequirementLevelOptIn,
					},
					"required_string_attr": {
						Description: "A required attribute with a string value",
						Type: ValueType{
							ValueType: pcommon.ValueTypeStr,
						},
						FullName:         "required_string_attr",
						RequirementLevel: AttributeRequirementLevelRequired,
					},
				},
				Metrics: map[MetricName]Metric{
					"default.metric": {
						Signal: Signal{
							Enabled:               true,
							Description:           "Monotonic cumulative sum int metric enabled by default.",
							ExtendedDocumentation: "The metric will be become optional soon.",
							Stability:             component.StabilityLevelDeprecated,
							Warnings: Warnings{
								IfEnabledNotSet: "This metric will be disabled by default soon.",
							},
							Attributes: []AttributeName{"string_attr", "overridden_int_attr", "enum_attr", "slice_attr", "map_attr", "conditional_int_attr", "conditional_string_attr", "opt_in_bool_attr"},
						},
						Unit: strPtr("s"),
						Sum: &Sum{
							MetricValueType:        MetricValueType{pmetric.NumberDataPointValueTypeInt},
							AggregationTemporality: AggregationTemporality{Aggregation: pmetric.AggregationTemporalityCumulative},
							Mono:                   Mono{Monotonic: true},
						},
						Deprecated: &Deprecated{
							Since: "1.0.0",
							Note:  "This metric will be removed",
						},
					},
					"reaggregate.metric": {
						Signal: Signal{
							Enabled:     true,
							Description: "Metric for testing spatial reaggregation",
							Stability:   component.StabilityLevelBeta,
							Attributes:  []AttributeName{"string_attr", "boolean_attr"},
						},
						Unit: strPtr("1"),
						Gauge: &Gauge{
							MetricValueType: MetricValueType{pmetric.NumberDataPointValueTypeDouble},
						},
					},
					"reaggregate.metric.with_required": {
						Signal: Signal{
							Enabled:     true,
							Description: "Metric for testing spatial reaggregation with required attributes",
							Stability:   component.StabilityLevelBeta,
							Attributes:  []AttributeName{"required_string_attr", "string_attr", "boolean_attr"},
						},
						Unit: strPtr("1"),
						Gauge: &Gauge{
							MetricValueType: MetricValueType{pmetric.NumberDataPointValueTypeDouble},
						},
					},
					"system.cpu.time": {
						Signal: Signal{
							Enabled:               true,
							Stability:             component.StabilityLevelBeta,
							SemanticConvention:    &SemanticConvention{SemanticConventionRef: "https://github.com/open-telemetry/semantic-conventions/blob/v1.38.0/docs/system/system-metrics.md#metric-systemcputime"},
							Description:           "Monotonic cumulative sum int metric enabled by default.",
							ExtendedDocumentation: "The metric will be become optional soon.",
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
							Stability:   component.StabilityLevelDeprecated,
							Warnings: Warnings{
								IfConfigured: "This metric is deprecated and will be removed soon.",
							},
							Attributes: []AttributeName{"string_attr", "boolean_attr", "boolean_attr2", "conditional_string_attr"},
						},
						Deprecated: &Deprecated{
							Since: "1.0.0",
							Note:  "This metric will be removed",
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
							Stability:   component.StabilityLevelDeprecated,
							Warnings: Warnings{
								IfConfigured: "This metric is deprecated and will be removed soon.",
							},
							Attributes: []AttributeName{"string_attr", "boolean_attr"},
						},
						Deprecated: &Deprecated{
							Since: "1.0.0",
							Note:  "This metric will be removed",
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
							Stability:             component.StabilityLevelDeprecated,
							Warnings: Warnings{
								IfEnabled: "This metric is deprecated and will be removed soon.",
							},
						},
						Deprecated: &Deprecated{
							Since: "1.0.0",
							Note:  "This metric will be removed",
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
							Stability:   component.StabilityLevelDevelopment,
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
							Attributes: []AttributeName{"string_attr", "overridden_int_attr", "enum_attr", "slice_attr", "map_attr", "conditional_int_attr", "conditional_string_attr", "opt_in_bool_attr"},
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
							Attributes: []AttributeName{"string_attr", "boolean_attr", "boolean_attr2", "conditional_string_attr"},
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
								Stability:   component.StabilityLevelDeprecated,
								Description: "Number of times the batch was sent due to a size trigger",
							},
							Deprecated: &Deprecated{
								Since: "1.5.0",
								Note:  "This metric will be removed in favor of batch_send_trigger_size",
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
								Stability:   component.StabilityLevelAlpha,
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
								Stability:   component.StabilityLevelStable,
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
								Stability:             component.StabilityLevelAlpha,
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
								Stability:   component.StabilityLevelDevelopment,
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
				Tests:           Tests{Host: "newMdatagenNopHost()"},
				FeatureGates: []FeatureGate{
					{
						ID:           "receiver.sample.featuregate.example",
						Description:  "This is an example feature gate for testing mdatagen code generation.",
						Stage:        "alpha",
						FromVersion:  "v0.100.0",
						ReferenceURL: "https://github.com/open-telemetry/opentelemetry-collector/issues/12345",
					},
				},
			},
		},
		{
			name: "testdata/parent.yaml",
			want: Metadata{
				Type:                 "subcomponent",
				Parent:               "parentComponent",
				GeneratedPackageName: "metadata",
				ScopeName:            "go.opentelemetry.io/collector/cmd/mdatagen/internal/testdata",
				PackageName:          "go.opentelemetry.io/collector/cmd/mdatagen/internal/testdata",
				ShortFolderName:      "testdata",
				Tests:                Tests{Host: "newMdatagenNopHost()"},
			},
		},
		{
			name: "testdata/generated_package_name.yaml",
			want: Metadata{
				Type:                 "custom",
				GeneratedPackageName: "customname",
				ScopeName:            "go.opentelemetry.io/collector/cmd/mdatagen/internal/testdata",
				PackageName:          "go.opentelemetry.io/collector/cmd/mdatagen/internal/testdata",
				ShortFolderName:      "testdata",
				Tests:                Tests{Host: "newMdatagenNopHost()"},
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
				PackageName:          "go.opentelemetry.io/collector/cmd/mdatagen/internal/testdata",
				ShortFolderName:      "testdata",
				Tests:                Tests{Host: "newMdatagenNopHost()"},
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
			name:    "testdata/invalid_metric_stability.yaml",
			want:    Metadata{},
			wantErr: "decoding failed due to the following error(s):\n\n'metrics[default.metric]' decoding failed due to the following error(s):\n\n'stability' unsupported stability level: \"development42\"",
		},
		{
			name:    "testdata/invalid_metric_semconvref.yaml",
			want:    Metadata{},
			wantErr: "metric \"default.metric\": invalid semantic-conventions URL: want https://github.com/open-telemetry/semantic-conventions/blob/v1.37.2/*#metric-defaultmetric, got \"https://github.com/open-telemetry/semantic-conventions/blob/v1.38.0/docs/system/system-metrics.md#metric-systemcputime\"",
		},
		{
			name:    "testdata/no_metric_stability.yaml",
			want:    Metadata{},
			wantErr: "metric \"default.metric\": missing required field: `stability.level`",
		},
		{
			name:    "testdata/undeprecated_with_deprecation.yaml",
			want:    Metadata{},
			wantErr: "`stability` must be `deprecated` when specifying a `deprecated` field",
		},
		{
			name:    "testdata/~~this file doesn't exist~~.yaml",
			wantErr: "unable to read the file file:testdata/~~this file doesn't exist~~.yaml",
		},
		{
			name: "testdata/display_name.yaml",
			want: Metadata{
				Type:                 "test",
				DisplayName:          "Test Receiver",
				GeneratedPackageName: "metadata",
				ScopeName:            "go.opentelemetry.io/collector/cmd/mdatagen/internal/testdata",
				PackageName:          "go.opentelemetry.io/collector/cmd/mdatagen/internal/testdata",
				ShortFolderName:      "testdata",
				Tests:                Tests{Host: "newMdatagenNopHost()"},
				Status: &Status{
					Class: "receiver",
					Stability: map[component.StabilityLevel][]string{
						component.StabilityLevelBeta: {"logs"},
					},
				},
			},
		},
		{
			name: "testdata/no_display_name.yaml",
			want: Metadata{
				Type:                 "nodisplayname",
				DisplayName:          "",
				GeneratedPackageName: "metadata",
				ScopeName:            "go.opentelemetry.io/collector/cmd/mdatagen/internal/testdata",
				PackageName:          "go.opentelemetry.io/collector/cmd/mdatagen/internal/testdata",
				ShortFolderName:      "testdata",
				Tests:                Tests{Host: "newMdatagenNopHost()"},
				Status: &Status{
					Class: "receiver",
					Stability: map[component.StabilityLevel][]string{
						component.StabilityLevelBeta: {"logs"},
					},
				},
			},
		},
		{
			name: "testdata/with_description.yaml",
			want: Metadata{
				Type:                 "testdesc",
				DisplayName:          "Test Component",
				Description:          "This is a test component with a description.",
				GeneratedPackageName: "metadata",
				ScopeName:            "go.opentelemetry.io/collector/cmd/mdatagen/internal/testdata",
				PackageName:          "go.opentelemetry.io/collector/cmd/mdatagen/internal/testdata",
				ShortFolderName:      "testdata",
				Tests:                Tests{Host: "newMdatagenNopHost()"},
				Status: &Status{
					Class: "receiver",
					Stability: map[component.StabilityLevel][]string{
						component.StabilityLevelBeta: {"logs"},
					},
				},
			},
		},
		{
			name: "testdata/with_underscore_in_semconv_ref_anchor_tag.yaml",
			want: Metadata{
				Type:                 "metricreceiver",
				GeneratedPackageName: "metadata",
				SemConvVersion:       "1.38.0",
				ScopeName:            "go.opentelemetry.io/collector/cmd/mdatagen/internal/testdata",
				PackageName:          "go.opentelemetry.io/collector/cmd/mdatagen/internal/testdata",
				ShortFolderName:      "testdata",
				Tests:                Tests{Host: "newMdatagenNopHost()"},
				Status: &Status{
					Class: "receiver",
					Stability: map[component.StabilityLevel][]string{
						component.StabilityLevelDevelopment: {"logs"},
						component.StabilityLevelBeta:        {"traces"},
						component.StabilityLevelStable:      {"metrics"},
					},
					Distributions: []string{"contrib"},
					Warnings:      []string{"Any additional information that should be brought to the consumer's attention"},
				},
				Metrics: map[MetricName]Metric{
					"system.disk.io_time": {
						Signal: Signal{
							Enabled:     true,
							Description: "Time disk spent activated..",
							Stability:   component.StabilityLevelDevelopment,
							SemanticConvention: &SemanticConvention{
								SemanticConventionRef: "https://github.com/open-telemetry/semantic-conventions/blob/v1.38.0/docs/system/system-metrics.md#metric-systemdiskio_time",
							},
						},
						Unit: strPtr("s"),
						Sum: &Sum{
							AggregationTemporality: AggregationTemporality{Aggregation: pmetric.AggregationTemporalityCumulative},
							Mono:                   Mono{Monotonic: true},
							MetricValueType:        MetricValueType{ValueType: pmetric.NumberDataPointValueTypeDouble},
						},
					},
				},
			},
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
