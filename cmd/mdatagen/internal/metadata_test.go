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
		{
			name:    "testdata/invalid_entity_stability.yaml",
			wantErr: `unsupported stability level: "stable42"`,
		},
		{
			name:    "testdata/entity_relationships_bidirectional.yaml",
			wantErr: `duplicate relationship to target "k8s.replicaset" (only one relationship allowed between two entities)`,
		},
		{
			name:    "testdata/entity_relationships_empty_type.yaml",
			wantErr: `entity "k8s.pod": relationship type cannot be empty`,
		},
		{
			name:    "testdata/entity_relationships_empty_target.yaml",
			wantErr: `entity "k8s.pod": relationship target cannot be empty`,
		},
		{
			name:    "testdata/entity_relationships_undefined_target.yaml",
			wantErr: `entity "k8s.pod": relationship target "k8s.replicaset" does not exist`,
		},
		{
			name:    "testdata/entity_metric_missing_association.yaml",
			wantErr: `metric "host.cpu.time": entity is required when entities are defined`,
		},
		{
			name:    "testdata/entity_event_missing_association.yaml",
			wantErr: `event "host.restart": entity is required when entities are defined`,
		},
		{
			name:    "testdata/entity_undefined_reference.yaml",
			wantErr: `metric "host.cpu.time": entity refers to undefined entity type: undefined_entity`,
		},
		{
			name:    "testdata/entity_single_metric_missing_association.yaml",
			wantErr: `metric "host.cpu.time": entity is required when entities are defined`,
		},
		{
			name:    "testdata/entity_metrics_events_valid.yaml",
			wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadMetadata(tt.name)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
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

func TestValidateFeatureGates(t *testing.T) {
	tests := []struct {
		name        string
		featureGate FeatureGate
		wantErr     string
	}{
		{
			name: "valid alpha gate",
			featureGate: FeatureGate{
				ID:           "component.feature",
				Description:  "Test feature gate",
				Stage:        FeatureGateStageAlpha,
				FromVersion:  "v0.100.0",
				ReferenceURL: "https://example.com",
			},
		},
		{
			name: "valid stable gate with to_version",
			featureGate: FeatureGate{
				ID:           "component.stable",
				Description:  "Stable feature gate",
				Stage:        FeatureGateStageStable,
				FromVersion:  "v0.90.0",
				ToVersion:    "v0.95.0",
				ReferenceURL: "https://example.com",
			},
		},
		{
			name: "empty description",
			featureGate: FeatureGate{
				ID:          "component.feature",
				Stage:       FeatureGateStageAlpha,
				FromVersion: "v0.100.0",
			},
			wantErr: `description is required`,
		},
		{
			name: "invalid stage",
			featureGate: FeatureGate{
				ID:          "component.feature",
				Description: "Test feature",
				Stage:       "invalid",
				FromVersion: "v0.100.0",
			},
			wantErr: `invalid stage "invalid"`,
		},
		{
			name: "missing from_version",
			featureGate: FeatureGate{
				ID:          "component.feature",
				Description: "Test feature",
				Stage:       FeatureGateStageAlpha,
			},
			wantErr: `from_version is required`,
		},
		{
			name: "from_version without v prefix",
			featureGate: FeatureGate{
				ID:          "component.feature",
				Description: "Test feature",
				Stage:       FeatureGateStageAlpha,
				FromVersion: "0.100.0",
			},
			wantErr: `from_version "0.100.0" must start with 'v'`,
		},
		{
			name: "to_version without v prefix",
			featureGate: FeatureGate{
				ID:          "component.feature",
				Description: "Test feature",
				Stage:       FeatureGateStageStable,
				FromVersion: "v0.90.0",
				ToVersion:   "0.95.0",
			},
			wantErr: `to_version "0.95.0" must start with 'v'`,
		},
		{
			name: "stable gate missing to_version",
			featureGate: FeatureGate{
				ID:          "component.feature",
				Description: "Test feature",
				Stage:       FeatureGateStageStable,
				FromVersion: "v0.90.0",
			},
			wantErr: `to_version is required for stable stage gates`,
		},
		{
			name: "deprecated gate missing to_version",
			featureGate: FeatureGate{
				ID:          "component.feature",
				Description: "Test feature",
				Stage:       FeatureGateStageDeprecated,
				FromVersion: "v0.90.0",
			},
			wantErr: `to_version is required for deprecated stage gates`,
		},
		{
			name: "missing reference_url",
			featureGate: FeatureGate{
				ID:          "component.feature",
				Description: "Test feature",
				Stage:       FeatureGateStageAlpha,
				FromVersion: "v0.100.0",
			},
			wantErr: `reference_url is required`,
		},
		{
			name: "invalid characters in ID",
			featureGate: FeatureGate{
				ID:           "component.feature@invalid",
				Description:  "Test feature",
				Stage:        FeatureGateStageAlpha,
				FromVersion:  "v0.100.0",
				ReferenceURL: "https://example.com",
			},
			wantErr: `ID contains invalid characters`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := &Metadata{
				FeatureGates: []FeatureGate{tt.featureGate},
			}
			err := md.validateFeatureGates()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateFeatureGatesEmptyID(t *testing.T) {
	md := &Metadata{
		FeatureGates: []FeatureGate{
			{
				Description: "Test",
				Stage:       FeatureGateStageAlpha,
			},
		},
	}
	err := md.validateFeatureGates()
	require.Error(t, err)
	assert.ErrorContains(t, err, "ID cannot be empty")
}

func TestValidateFeatureGatesDuplicateID(t *testing.T) {
	md := &Metadata{
		FeatureGates: []FeatureGate{
			{
				ID:          "component.feature",
				Description: "Test feature",
				Stage:       FeatureGateStageAlpha,
			},
			{
				ID:          "component.feature",
				Description: "Duplicate feature",
				Stage:       FeatureGateStageAlpha,
			},
		},
	}
	err := md.validateFeatureGates()
	require.Error(t, err)
	assert.ErrorContains(t, err, "duplicate ID")
}

func TestValidateFeatureGatesNotSorted(t *testing.T) {
	md := &Metadata{
		FeatureGates: []FeatureGate{
			{
				ID:           "component.zebra",
				Description:  "Test feature",
				Stage:        FeatureGateStageAlpha,
				ReferenceURL: "https://example.com",
			},
			{
				ID:           "component.alpha",
				Description:  "Another feature",
				Stage:        FeatureGateStageAlpha,
				ReferenceURL: "https://example.com",
			},
		},
	}
	err := md.validateFeatureGates()
	require.Error(t, err)
	assert.ErrorContains(t, err, "feature gates must be sorted by ID")
}

func TestInferSemConvFromMetricName(t *testing.T) {
	tests := []struct {
		name     string
		metric   string
		wantPkg  string
		wantType string
		wantErr  bool
	}{
		{
			name:     "go.goroutine.count metric returns correct package and type",
			metric:   "go.goroutine.count",
			wantPkg:  "goconv",
			wantType: "GoroutineCount",
		},
		{
			name:     "k8s.container.cpu.limit metric returns correct package and type",
			metric:   "k8s.container.cpu.limit",
			wantPkg:  "k8sconv",
			wantType: "ContainerCPULimit",
		},
		{
			name:     "system.cpu.time metric returns correct package and type",
			metric:   "system.cpu.time",
			wantPkg:  "systemconv",
			wantType: "CPUTime",
		},
		{
			name:     "system.cpu.logical.count metric returns correct package and type",
			metric:   "system.cpu.logical.count",
			wantPkg:  "systemconv",
			wantType: "CPULogicalCount",
		},
		{
			name:     "system.memory.limit metric returns correct package and type",
			metric:   "system.memory.limit",
			wantPkg:  "systemconv",
			wantType: "MemoryLimit",
		},
		{
			name:     "system.disk.io uses IO acronym",
			metric:   "system.disk.io",
			wantPkg:  "systemconv",
			wantType: "DiskIO",
		},
		{
			name:     "system.disk.io_time handles underscores and IO",
			metric:   "system.disk.io_time",
			wantPkg:  "systemconv",
			wantType: "DiskIOTime",
		},
		{
			name:     "system.network.io metric returns correct package and type",
			metric:   "system.network.io",
			wantPkg:  "systemconv",
			wantType: "NetworkIO",
		},
		{
			name:     "system.linux.memory.available metric returns correct package and type",
			metric:   "system.linux.memory.available",
			wantPkg:  "systemconv",
			wantType: "LinuxMemoryAvailable",
		},
		{
			name:     "system.linux.memory.slab.usage metric returns correct package and type",
			metric:   "system.linux.memory.slab.usage",
			wantPkg:  "systemconv",
			wantType: "LinuxMemorySlabUsage",
		},
		{
			name:     "system.uptime single remaining segment",
			metric:   "system.uptime",
			wantPkg:  "systemconv",
			wantType: "Uptime",
		},
		{
			name:     "system.filesystem.utilization metric returns correct package and type",
			metric:   "system.filesystem.utilization",
			wantPkg:  "systemconv",
			wantType: "FilesystemUtilization",
		},
		{
			name:     "system.paging.faults metric returns correct package and type",
			metric:   "system.paging.faults",
			wantPkg:  "systemconv",
			wantType: "PagingFaults",
		},
		{
			name:     "system.network.packet.dropped metric returns correct package and type",
			metric:   "system.network.packet.dropped",
			wantPkg:  "systemconv",
			wantType: "NetworkPacketDropped",
		},
		{
			name:     "http.server.request.duration uses different namespace",
			metric:   "http.server.request.duration",
			wantPkg:  "httpconv",
			wantType: "ServerRequestDuration",
		},
		{
			name:    "single segment fails",
			metric:  "system",
			wantErr: true,
		},
		{
			name:    "empty string fails",
			metric:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, typeName, err := InferSemConvFromMetricName(tt.metric)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPkg, pkg)
			assert.Equal(t, tt.wantType, typeName)
		})
	}
}

func TestInferSemConvTypes(t *testing.T) {
	tests := []struct {
		name       string
		metricName string
		md         Metadata
		wantPkg    string
		wantType   string
		wantNil    bool
	}{
		{
			name:       "system.cpu.time has valid package and type",
			metricName: "system.cpu.time",
			md: Metadata{
				Metrics: map[MetricName]Metric{
					"system.cpu.time": {
						Signal: Signal{
							SemanticConvention: &SemanticConvention{
								SemanticConventionRef: "https://example.com",
							},
						},
					},
				},
			},
			wantPkg:  "systemconv",
			wantType: "CPUTime",
		},
		{
			name:       "system.disk.io has valid package and type",
			metricName: "system.disk.io",
			md: Metadata{
				Metrics: map[MetricName]Metric{
					"system.disk.io": {
						Signal: Signal{
							SemanticConvention: &SemanticConvention{
								SemanticConventionRef: "https://example.com",
							},
						},
					},
				},
			},
			wantPkg:  "systemconv",
			wantType: "DiskIO",
		},
		{
			name:       "no semantic convention doesn't do anything",
			metricName: "default.metric",
			md: Metadata{
				Metrics: map[MetricName]Metric{
					"default.metric": {
						Signal: Signal{},
					},
				},
			},
			wantNil: true,
		},
		{
			name:       "invalid metric name doesn't pupulate pkg and type",
			metricName: "invalid",
			md: Metadata{
				Metrics: map[MetricName]Metric{
					"invalid": {
						Signal: Signal{
							SemanticConvention: &SemanticConvention{
								SemanticConventionRef: "https://example.com",
							},
						},
					},
				},
			},
			wantPkg:  "",
			wantType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.md.inferSemConvTypes()
			sc := tt.md.Metrics[MetricName(tt.metricName)].SemanticConvention
			if tt.wantNil {
				assert.Nil(t, sc)
				require.Nil(t, sc, "The semantic convention must be nil")
			} else {
				require.NotNil(t, sc, "The semantic convention must not be nil")
				assert.Equal(t, tt.wantPkg, sc.Package)
				assert.Equal(t, tt.wantType, sc.Type)
			}
		})
	}
}

func TestSemConvImports(t *testing.T) {
	tests := []struct {
		name string
		md   Metadata
		want []SemConvImport
	}{
		{
			name: "empty semconv import",
			md:   Metadata{},
			want: []SemConvImport{},
		},
		{
			name: "Multiple packages sorted alphabetically",
			md: Metadata{
				Metrics: map[MetricName]Metric{
					"system.cpu.time": {
						Signal: Signal{
							SemanticConvention: &SemanticConvention{
								Package: "systemconv",
							},
						},
					},
					"http.server.duration": {
						Signal: Signal{
							SemanticConvention: &SemanticConvention{
								Package: "httpconv",
							},
						},
					},
				},
			},
			want: []SemConvImport{
				{
					Package: "httpconv",
					Alias:   "httpconv",
				},
				{
					Package: "systemconv",
					Alias:   "systemconv",
				},
			},
		},
		{
			name: "duplicate packages deduplicates",
			md: Metadata{
				Metrics: map[MetricName]Metric{
					"system.cpu.time": {
						Signal: Signal{
							SemanticConvention: &SemanticConvention{
								Package: "systemconv",
							},
						},
					},
					"system.memory.usage": {
						Signal: Signal{
							SemanticConvention: &SemanticConvention{
								Package: "systemconv",
							},
						},
					},
				},
			},
			want: []SemConvImport{
				{
					Package: "systemconv",
					Alias:   "systemconv",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.md.SemConvImports()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestShouldUseSemConvValues(t *testing.T) {
	tests := []struct {
		name string
		sc   *SemanticConvention
		want bool
	}{
		{
			name: "nil semantic convention",
			sc:   nil,
			want: false,
		},
		{
			name: "empty type",
			sc:   &SemanticConvention{Type: ""},
			want: false,
		},
		{
			name: "type set returns true",
			sc:   &SemanticConvention{Type: "CPUTime"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.sc.ShouldUseSemConvValues()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestImportPath(t *testing.T) {
	tests := []struct {
		name string
		sc   *SemanticConvention
		scv  string
		want string
	}{
		{
			name: "Empty package returns empty string",
			sc: &SemanticConvention{
				Package: "",
			},
			scv:  "1.38.0",
			want: "",
		},
		{
			name: "systemconv package",
			sc: &SemanticConvention{
				Package: "systemconv",
			},
			scv:  "1.38.0",
			want: "go.opentelemetry.io/otel/semconv/1.38.0/systemconv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.sc.ImportPath(tt.scv)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatSemConvIdentifiers(t *testing.T) {
	tests := []struct {
		name              string
		semconvIdentifier string
		want              string
		wantErr           bool
	}{
		{
			name:              "correctly formats semconv go type",
			semconvIdentifier: "cpu.time",
			want:              "CPUTime",
		},
		{
			name:              "correctly formats disk.io to DiskIO",
			semconvIdentifier: "disk.io",
			want:              "DiskIO",
		},
		{
			name:              "returns an error when an empty string is supplied",
			semconvIdentifier: "",
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatSemConvIdentifier(tt.semconvIdentifier)
			if tt.wantErr {
				require.Error(t, err, "error is required for empty strings")
				assert.Empty(t, got, "expect an empty string if an error is returned")
			} else {
				require.NoError(t, err, "require no error")
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
