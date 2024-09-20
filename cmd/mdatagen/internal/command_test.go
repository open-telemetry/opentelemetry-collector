// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

func TestRunContents(t *testing.T) {
	tests := []struct {
		yml                    string
		wantMetricsGenerated   bool
		wantMetricsContext     bool
		wantConfigGenerated    bool
		wantTelemetryGenerated bool
		wantStatusGenerated    bool
		wantGoleakIgnore       bool
		wantGoleakSkip         bool
		wantGoleakSetup        bool
		wantGoleakTeardown     bool
		wantErr                bool
		customPackageName      string
	}{
		{
			yml:     "invalid.yaml",
			wantErr: true,
		},
		{
			yml:                  "metrics_and_type.yaml",
			wantMetricsGenerated: true,
			wantConfigGenerated:  true,
			wantStatusGenerated:  true,
		},
		{
			yml:                 "resource_attributes_only.yaml",
			wantConfigGenerated: true,
			wantStatusGenerated: true,
		},
		{
			yml:                 "status_only.yaml",
			wantStatusGenerated: true,
		},
		{
			yml:                 "with_tests_receiver.yaml",
			wantStatusGenerated: true,
		},
		{
			yml:                 "with_tests_exporter.yaml",
			wantStatusGenerated: true,
		},
		{
			yml:                 "with_tests_processor.yaml",
			wantStatusGenerated: true,
		},
		{
			yml:                 "with_tests_extension.yaml",
			wantStatusGenerated: true,
		},
		{
			yml:                 "with_tests_connector.yaml",
			wantStatusGenerated: true,
		},
		{
			yml:                 "with_goleak_ignores.yaml",
			wantStatusGenerated: true,
			wantGoleakIgnore:    true,
		},
		{
			yml:                 "with_goleak_skip.yaml",
			wantStatusGenerated: true,
			wantGoleakSkip:      true,
		},
		{
			yml:                 "with_goleak_setup.yaml",
			wantStatusGenerated: true,
			wantGoleakSetup:     true,
		},
		{
			yml:                 "with_goleak_teardown.yaml",
			wantStatusGenerated: true,
			wantGoleakTeardown:  true,
		},
		{
			yml:                    "with_telemetry.yaml",
			wantStatusGenerated:    true,
			wantTelemetryGenerated: true,
		},
		{
			yml:     "invalid_telemetry_missing_value_type_for_histogram.yaml",
			wantErr: true,
		},
		{
			yml:                  "async_metric.yaml",
			wantMetricsGenerated: true,
			wantConfigGenerated:  true,
			wantStatusGenerated:  true,
		},
		{
			yml:                  "metrics_and_type.yaml",
			wantMetricsGenerated: true,
			wantConfigGenerated:  true,
			wantStatusGenerated:  true,
			customPackageName:    "custom",
		},
	}
	for _, tt := range tests {
		t.Run(tt.yml, func(t *testing.T) {
			tmpdir := filepath.Join(t.TempDir(), "shortname")
			err := os.MkdirAll(tmpdir, 0750)
			require.NoError(t, err)
			ymlContent, err := os.ReadFile(filepath.Join("testdata", tt.yml))
			require.NoError(t, err)
			metadataFile := filepath.Join(tmpdir, "metadata.yaml")
			require.NoError(t, os.WriteFile(metadataFile, ymlContent, 0600))
			require.NoError(t, os.WriteFile(filepath.Join(tmpdir, "README.md"), []byte(`
<!-- status autogenerated section -->
foo
<!-- end autogenerated section -->`), 0600))

			generatedPackageName := tt.customPackageName
			if generatedPackageName == "" {
				generatedPackageName = "metadata"
			}

			err = run(metadataFile, generatedPackageName)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			generatedPackageDir := filepath.Join("internal", generatedPackageName)

			var contents []byte
			if tt.wantMetricsGenerated {
				require.FileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_metrics.go"))
				require.FileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_metrics_test.go"))
				require.FileExists(t, filepath.Join(tmpdir, "documentation.md"))
				contents, err = os.ReadFile(filepath.Join(tmpdir, generatedPackageDir, "generated_metrics.go")) // nolint: gosec
				require.NoError(t, err)
				if tt.wantMetricsContext {
					require.Contains(t, string(contents), "\"context\"")
				} else {
					require.NotContains(t, string(contents), "\"context\"")
				}
			} else {
				require.NoFileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_metrics.go"))
				require.NoFileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_metrics_test.go"))
			}

			if tt.wantConfigGenerated {
				require.FileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_config.go"))
				require.FileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_config_test.go"))
			} else {
				require.NoFileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_config.go"))
				require.NoFileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_config_test.go"))
			}

			if tt.wantTelemetryGenerated {
				require.FileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_telemetry.go"))
				require.FileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_telemetry_test.go"))
				require.FileExists(t, filepath.Join(tmpdir, "documentation.md"))
				contents, err = os.ReadFile(filepath.Join(tmpdir, generatedPackageDir, "generated_telemetry.go")) // nolint: gosec
				require.NoError(t, err)
				if tt.wantMetricsContext {
					require.Contains(t, string(contents), "\"context\"")
				} else {
					require.NotContains(t, string(contents), "\"context\"")
				}
			} else {
				require.NoFileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_telemetry.go"))
				require.NoFileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_telemetry_test.go"))
			}

			if !tt.wantMetricsGenerated && !tt.wantTelemetryGenerated {
				require.NoFileExists(t, filepath.Join(tmpdir, "documentation.md"))
			}

			if tt.wantStatusGenerated {
				require.FileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_status.go"))
				contents, err = os.ReadFile(filepath.Join(tmpdir, "README.md")) // nolint: gosec
				require.NoError(t, err)
				require.NotContains(t, string(contents), "foo")
			} else {
				require.NoFileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_status.go"))
				contents, err = os.ReadFile(filepath.Join(tmpdir, "README.md")) // nolint: gosec
				require.NoError(t, err)
				require.Contains(t, string(contents), "foo")
			}

			require.FileExists(t, filepath.Join(tmpdir, "generated_component_test.go"))
			contents, err = os.ReadFile(filepath.Join(tmpdir, "generated_component_test.go")) // nolint: gosec
			require.NoError(t, err)
			require.Contains(t, string(contents), "func Test")
			_, err = parser.ParseFile(token.NewFileSet(), "", contents, parser.DeclarationErrors)
			require.NoError(t, err)

			require.FileExists(t, filepath.Join(tmpdir, "generated_package_test.go"))
			contents, err = os.ReadFile(filepath.Join(tmpdir, "generated_package_test.go")) // nolint: gosec
			require.NoError(t, err)
			require.Contains(t, string(contents), "func TestMain")
			_, err = parser.ParseFile(token.NewFileSet(), "", contents, parser.DeclarationErrors)
			require.NoError(t, err)

			if tt.wantGoleakSkip {
				require.Contains(t, string(contents), "skipping goleak test")
			} else {
				require.NotContains(t, string(contents), "skipping goleak test")
			}

			if tt.wantGoleakIgnore {
				require.Contains(t, string(contents), "IgnoreTopFunction")
				require.Contains(t, string(contents), "IgnoreAnyFunction")
			} else {
				require.NotContains(t, string(contents), "IgnoreTopFunction")
				require.NotContains(t, string(contents), "IgnoreAnyFunction")
			}

			if tt.wantGoleakSetup {
				require.Contains(t, string(contents), "setupFunc")
			} else {
				require.NotContains(t, string(contents), "setupFunc")
			}

			if tt.wantGoleakTeardown {
				require.Contains(t, string(contents), "teardownFunc")
			} else {
				require.NotContains(t, string(contents), "teardownFunc")
			}
		})
	}
}

func TestRun(t *testing.T) {
	type args struct {
		ymlPath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "no argument",
			args:    args{""},
			wantErr: true,
		},
		{
			name:    "no such file",
			args:    args{"/no/such/file"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := run(tt.args.ymlPath, "metadata"); (err != nil) != tt.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInlineReplace(t *testing.T) {
	tests := []struct {
		name           string
		markdown       string
		outputFile     string
		componentClass string
		warnings       []string
		stability      map[component.StabilityLevel][]string
		distros        []string
		codeowners     *Codeowners
		githubProject  string
	}{
		{
			name: "readme with empty status",
			markdown: `# Some component

<!-- status autogenerated section -->
<!-- end autogenerated section -->

Some info about a component
`,
			outputFile:     "readme_with_status.md",
			componentClass: "receiver",
			distros:        []string{"contrib"},
			githubProject:  "open-telemetry/opentelemetry-collector",
		},
		{
			name: "readme with status for extension",
			markdown: `# Some component

<!-- status autogenerated section -->
<!-- end autogenerated section -->

Some info about a component
`,
			outputFile:     "readme_with_status_extension.md",
			componentClass: "extension",
			distros:        []string{"contrib"},
		},
		{
			name: "readme with status with codeowners and seeking new",
			markdown: `# Some component

<!-- status autogenerated section -->
<!-- end autogenerated section -->

Some info about a component
`,
			outputFile:     "readme_with_status_codeowners_and_seeking_new.md",
			componentClass: "receiver",
			distros:        []string{"contrib"},
			codeowners: &Codeowners{
				Active:     []string{"foo"},
				SeekingNew: true,
			},
		},
		{
			name: "readme with status with codeowners and emeritus",
			markdown: `# Some component

<!-- status autogenerated section -->
<!-- end autogenerated section -->

Some info about a component
`,
			outputFile:     "readme_with_status_codeowners_and_emeritus.md",
			componentClass: "receiver",
			distros:        []string{"contrib"},
			codeowners: &Codeowners{
				Active:   []string{"foo"},
				Emeritus: []string{"bar"},
			},
		},
		{
			name: "readme with status with codeowners",
			markdown: `# Some component

<!-- status autogenerated section -->
<!-- end autogenerated section -->

Some info about a component
`,
			outputFile:     "readme_with_status_codeowners.md",
			componentClass: "receiver",
			distros:        []string{"contrib"},
			codeowners: &Codeowners{
				Active: []string{"foo"},
			},
		},
		{
			name: "readme with status table",
			markdown: `# Some component

<!-- status autogenerated section -->
| Status                   |           |
| ------------------------ |-----------|
<!-- end autogenerated section -->

Some info about a component
`,
			outputFile:     "readme_with_status.md",
			componentClass: "receiver",
			distros:        []string{"contrib"},
			githubProject:  "open-telemetry/opentelemetry-collector",
		},
		{
			name: "readme with no status",
			markdown: `# Some component

Some info about a component
`,
			outputFile: "readme_without_status.md",
			distros:    []string{"contrib"},
		},
		{
			name: "component with warnings",
			markdown: `# Some component

<!-- status autogenerated section -->
<!-- end autogenerated section -->

Some info about a component
### warnings
Some warning there.
`,
			outputFile: "readme_with_warnings.md",
			warnings:   []string{"warning1"},
			distros:    []string{"contrib"},
		},
		{
			name: "readme with multiple signals",
			markdown: `# Some component

<!-- status autogenerated section -->
<!-- end autogenerated section -->

Some info about a component
`,
			outputFile: "readme_with_multiple_signals.md",
			stability: map[component.StabilityLevel][]string{
				component.StabilityLevelBeta:  {"metrics"},
				component.StabilityLevelAlpha: {"logs"},
			},
			distros: []string{"contrib"},
		},
		{
			name: "readme with cmd class",
			markdown: `# Some component

<!-- status autogenerated section -->
<!-- end autogenerated section -->

Some info about a component
`,
			outputFile: "readme_with_cmd_class.md",
			stability: map[component.StabilityLevel][]string{
				component.StabilityLevelBeta:  {"metrics"},
				component.StabilityLevelAlpha: {"logs"},
			},
			componentClass: "cmd",
			distros:        []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stability := map[component.StabilityLevel][]string{component.StabilityLevelBeta: {"metrics"}}
			if len(tt.stability) > 0 {
				stability = tt.stability
			}
			md := Metadata{
				GithubProject:   tt.githubProject,
				Type:            "foo",
				ShortFolderName: "foo",
				Status: &Status{
					Stability:     stability,
					Distributions: tt.distros,
					Class:         tt.componentClass,
					Warnings:      tt.warnings,
					Codeowners:    tt.codeowners,
				},
			}
			tmpdir := t.TempDir()

			readmeFile := filepath.Join(tmpdir, "README.md")
			require.NoError(t, os.WriteFile(readmeFile, []byte(tt.markdown), 0600))

			err := inlineReplace("templates/readme.md.tmpl", readmeFile, md, statusStart, statusEnd, "metadata")
			require.NoError(t, err)

			require.FileExists(t, filepath.Join(tmpdir, "README.md"))
			got, err := os.ReadFile(filepath.Join(tmpdir, "README.md")) // nolint: gosec
			require.NoError(t, err)
			got = bytes.ReplaceAll(got, []byte("\r\n"), []byte("\n"))
			expected, err := os.ReadFile(filepath.Join("testdata", tt.outputFile))
			require.NoError(t, err)
			expected = bytes.ReplaceAll(expected, []byte("\r\n"), []byte("\n"))
			fmt.Println(string(got))
			fmt.Println(string(expected))
			require.Equal(t, string(expected), string(got))
		})
	}
}

func TestGenerateStatusMetadata(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		md       Metadata
		expected string
	}{
		{
			name: "foo component with beta status",
			md: Metadata{
				Type: "foo",
				Status: &Status{
					Stability: map[component.StabilityLevel][]string{
						component.StabilityLevelBeta: {"metrics"},
					},
					Distributions: []string{"contrib"},
					Class:         "receiver",
				},
			},
			expected: `// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"go.opentelemetry.io/collector/component"
)

var (
	Type      = component.MustNewType("foo")
	ScopeName = ""
)

const (
	MetricsStability = component.StabilityLevelBeta
)
`,
		},
		{
			name: "foo component with alpha status",
			md: Metadata{
				Type: "foo",
				Status: &Status{
					Stability: map[component.StabilityLevel][]string{
						component.StabilityLevelAlpha: {"metrics"},
					},
					Distributions: []string{"contrib"},
					Class:         "receiver",
				},
			},
			expected: `// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"go.opentelemetry.io/collector/component"
)

var (
	Type      = component.MustNewType("foo")
	ScopeName = ""
)

const (
	MetricsStability = component.StabilityLevelAlpha
)
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			err := generateFile("templates/status.go.tmpl",
				filepath.Join(tmpdir, "generated_status.go"), tt.md, "metadata")
			require.NoError(t, err)
			actual, err := os.ReadFile(filepath.Join(tmpdir, "generated_status.go")) // nolint: gosec
			require.NoError(t, err)
			require.Equal(t, tt.expected, string(actual))
		})
	}
}

func TestGenerateTelemetryMetadata(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		md       Metadata
		expected string
	}{
		{
			name: "foo component with beta status",
			md: Metadata{
				Type: "foo",
				Status: &Status{
					Stability: map[component.StabilityLevel][]string{
						component.StabilityLevelBeta: {"metrics"},
					},
					Distributions: []string{"contrib"},
					Class:         "receiver",
				},
			},
			expected: `// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
)

// Deprecated: [v0.108.0] use LeveledMeter instead.
func Meter(settings component.TelemetrySettings) metric.Meter {
	return settings.MeterProvider.Meter("")
}

func LeveledMeter(settings component.TelemetrySettings, level configtelemetry.Level) metric.Meter {
	return settings.LeveledMeterProvider(level).Meter("")
}

func Tracer(settings component.TelemetrySettings) trace.Tracer {
	return settings.TracerProvider.Tracer("")
}
`,
		},
		{
			name: "foo component with alpha status",
			md: Metadata{
				Type: "foo",
				Status: &Status{
					Stability: map[component.StabilityLevel][]string{
						component.StabilityLevelAlpha: {"metrics"},
					},
					Distributions: []string{"contrib"},
					Class:         "receiver",
				},
			},
			expected: `// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
)

// Deprecated: [v0.108.0] use LeveledMeter instead.
func Meter(settings component.TelemetrySettings) metric.Meter {
	return settings.MeterProvider.Meter("")
}

func LeveledMeter(settings component.TelemetrySettings, level configtelemetry.Level) metric.Meter {
	return settings.LeveledMeterProvider(level).Meter("")
}

func Tracer(settings component.TelemetrySettings) trace.Tracer {
	return settings.TracerProvider.Tracer("")
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			err := generateFile("templates/telemetry.go.tmpl",
				filepath.Join(tmpdir, "generated_telemetry.go"), tt.md, "metadata")
			require.NoError(t, err)
			actual, err := os.ReadFile(filepath.Join(tmpdir, "generated_telemetry.go")) // nolint: gosec
			require.NoError(t, err)
			require.Equal(t, tt.expected, string(actual))
		})
	}
}
