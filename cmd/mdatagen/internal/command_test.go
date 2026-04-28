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
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"
	"go.opentelemetry.io/collector/component"
)

func TestNewCommand(t *testing.T) {
	cmd, err := NewCommand()
	require.NoError(t, err)

	assert.NotNil(t, cmd)
	assert.IsType(t, &cobra.Command{}, cmd)
	assert.Equal(t, "mdatagen", cmd.Use)
	assert.True(t, cmd.SilenceUsage)
}

func TestCommandNoArgs(t *testing.T) {
	cmd, err := NewCommand()
	require.NoError(t, err)

	cmd.SetArgs([]string{})
	err = cmd.Execute()

	require.Error(t, err)
}

func TestCommandErrorOutputOnce(t *testing.T) {
	cmd, err := NewCommand()
	require.NoError(t, err)

	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"/nonexistent/path/metadata.yaml"})

	err = cmd.Execute()
	require.Error(t, err)
	out := stderr.String()
	require.NotEmpty(t, out)

	msg := err.Error()
	assert.Equal(t, 1, strings.Count(out, msg), out)
}

func TestRunContents(t *testing.T) {
	tests := []struct {
		yml                  string
		wantMetricsGenerated bool
		// TODO: we should add one more flag for logs builder
		wantEventsGenerated             bool
		wantMetricsContext              bool
		wantLogsGenerated               bool
		wantConfigGenerated             bool
		wantTelemetryGenerated          bool
		wantResourceAttributesGenerated bool
		wantReadmeGenerated             bool
		wantStatusGenerated             bool
		wantComponentTestGenerated      bool
		wantGoleakIgnore                bool
		wantGoleakSkip                  bool
		wantGoleakSetup                 bool
		wantGoleakTeardown              bool
		wantFeatureGatesGenerated       bool
		wantConfigSchemaGenerated       bool
		wantMetricsSchemaYamlGenerated  bool
		wantErr                         bool
		wantOrderErr                    bool
		wantRunErr                      bool
		wantAttributes                  []string
	}{
		{
			yml:     "invalid.yaml",
			wantErr: true,
		},
		{
			yml:          "unsorted_rattr.yaml",
			wantOrderErr: true,
		},
		{
			yml:                        "basic_connector.yaml",
			wantErr:                    false,
			wantStatusGenerated:        true,
			wantReadmeGenerated:        true,
			wantComponentTestGenerated: true,
		},
		{
			yml:                        "basic_receiver.yaml",
			wantErr:                    false,
			wantStatusGenerated:        true,
			wantReadmeGenerated:        true,
			wantComponentTestGenerated: true,
			wantLogsGenerated:          true,
		},
		{
			yml:                 "basic_pkg.yaml",
			wantErr:             false,
			wantStatusGenerated: false,
			wantReadmeGenerated: true,
		},
		{
			yml:                            "metrics_and_type.yaml",
			wantMetricsGenerated:           true,
			wantConfigGenerated:            true,
			wantStatusGenerated:            true,
			wantReadmeGenerated:            true,
			wantComponentTestGenerated:     true,
			wantMetricsSchemaYamlGenerated: true,
		},
		{
			yml:                             "resource_attributes_only.yaml",
			wantConfigGenerated:             true,
			wantStatusGenerated:             true,
			wantResourceAttributesGenerated: true,
			wantReadmeGenerated:             true,
			wantComponentTestGenerated:      true,
			wantLogsGenerated:               true,
			wantMetricsSchemaYamlGenerated:  true,
		},
		{
			yml:                        "status_only.yaml",
			wantStatusGenerated:        true,
			wantReadmeGenerated:        true,
			wantComponentTestGenerated: true,
		},
		{
			yml:                        "with_tests_receiver.yaml",
			wantStatusGenerated:        true,
			wantReadmeGenerated:        true,
			wantComponentTestGenerated: true,
			wantLogsGenerated:          true,
		},
		{
			yml:                        "with_tests_exporter.yaml",
			wantStatusGenerated:        true,
			wantReadmeGenerated:        true,
			wantComponentTestGenerated: true,
		},
		{
			yml:                        "with_tests_processor.yaml",
			wantStatusGenerated:        true,
			wantReadmeGenerated:        true,
			wantComponentTestGenerated: true,
		},
		{
			yml:                        "with_tests_extension.yaml",
			wantStatusGenerated:        true,
			wantReadmeGenerated:        true,
			wantComponentTestGenerated: true,
		},
		{
			yml:                        "with_tests_connector.yaml",
			wantStatusGenerated:        true,
			wantReadmeGenerated:        true,
			wantComponentTestGenerated: true,
		},
		{
			yml:                        "with_tests_profiles_connector.yaml",
			wantStatusGenerated:        true,
			wantReadmeGenerated:        true,
			wantComponentTestGenerated: true,
		},
		{
			yml:                        "with_goleak_ignores.yaml",
			wantStatusGenerated:        true,
			wantGoleakIgnore:           true,
			wantReadmeGenerated:        true,
			wantComponentTestGenerated: true,
		},
		{
			yml:                        "with_goleak_skip.yaml",
			wantStatusGenerated:        true,
			wantGoleakSkip:             true,
			wantReadmeGenerated:        true,
			wantComponentTestGenerated: true,
		},
		{
			yml:                        "with_goleak_setup.yaml",
			wantStatusGenerated:        true,
			wantGoleakSetup:            true,
			wantReadmeGenerated:        true,
			wantComponentTestGenerated: true,
		},
		{
			yml:                        "with_goleak_teardown.yaml",
			wantStatusGenerated:        true,
			wantGoleakTeardown:         true,
			wantReadmeGenerated:        true,
			wantComponentTestGenerated: true,
		},
		{
			yml:                        "with_telemetry.yaml",
			wantStatusGenerated:        true,
			wantTelemetryGenerated:     true,
			wantReadmeGenerated:        true,
			wantComponentTestGenerated: true,
			wantAttributes:             []string{"name"},
			wantLogsGenerated:          true,
		},
		{
			yml:                        "invalid_telemetry_missing_value_type_for_histogram.yaml",
			wantErr:                    true,
			wantReadmeGenerated:        true,
			wantComponentTestGenerated: true,
		},
		{
			yml:                            "async_metric.yaml",
			wantMetricsGenerated:           true,
			wantConfigGenerated:            true,
			wantStatusGenerated:            true,
			wantReadmeGenerated:            true,
			wantComponentTestGenerated:     true,
			wantMetricsSchemaYamlGenerated: true,
		},
		{
			yml:                        "custom_generated_package_name.yaml",
			wantStatusGenerated:        true,
			wantReadmeGenerated:        true,
			wantComponentTestGenerated: true,
			wantLogsGenerated:          true,
		},
		{
			yml:                        "feature_gates.yaml",
			wantStatusGenerated:        true,
			wantReadmeGenerated:        true,
			wantComponentTestGenerated: true,
			wantFeatureGatesGenerated:  true,
		},
		{
			yml:                            "with_conditional_attribute.yaml",
			wantStatusGenerated:            true,
			wantReadmeGenerated:            true,
			wantMetricsGenerated:           true,
			wantLogsGenerated:              true,
			wantConfigGenerated:            true,
			wantComponentTestGenerated:     true,
			wantMetricsSchemaYamlGenerated: true,
		},
		{
			yml:                            "events/basic_event.yaml",
			wantStatusGenerated:            true,
			wantReadmeGenerated:            true,
			wantComponentTestGenerated:     true,
			wantConfigGenerated:            true,
			wantEventsGenerated:            true,
			wantLogsGenerated:              true,
			wantMetricsSchemaYamlGenerated: true,
		},
		{
			yml:                        "with_config.yaml",
			wantStatusGenerated:        true,
			wantReadmeGenerated:        true,
			wantLogsGenerated:          true,
			wantComponentTestGenerated: true,
			wantConfigSchemaGenerated:  true,
		},
		{
			yml:        "with_invalid_config_ref.yaml",
			wantRunErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.yml, func(t *testing.T) {
			tmpdir := filepath.Join(t.TempDir(), "shortname")
			err := os.MkdirAll(tmpdir, 0o750)
			require.NoError(t, err)
			ymlContent, err := os.ReadFile(filepath.Join("testdata", tt.yml))
			require.NoError(t, err)
			metadataFile := filepath.Join(tmpdir, "metadata.yaml")
			require.NoError(t, os.WriteFile(metadataFile, ymlContent, 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(tmpdir, "empty.go"), []byte("package shortname"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(tmpdir, "go.mod"), []byte("module shortname"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(tmpdir, "README.md"), []byte(`
<!-- status autogenerated section -->
foo
<!-- end autogenerated section -->`), 0o600))
			md, err := LoadMetadata(metadataFile)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			generatedPackageDir := filepath.Join("internal", md.GeneratedPackageName)
			require.NoError(t, os.MkdirAll(filepath.Join(tmpdir, generatedPackageDir), 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(tmpdir, generatedPackageDir, "generated_status.go"), []byte("status"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(tmpdir, generatedPackageDir, "generated_telemetry_test.go"), []byte("test"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(tmpdir, generatedPackageDir, "generated_component_test.go"), []byte("test"), 0o600))

			err = run(metadataFile)
			if tt.wantOrderErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "metadata.yaml ordering check failed")
				return
			}
			if tt.wantRunErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to generate config files")
				return
			}
			require.NoError(t, err)

			// Documentation is generated when any of these features are present
			wantDocumentationGenerated := tt.wantFeatureGatesGenerated || tt.wantMetricsGenerated || tt.wantTelemetryGenerated || tt.wantResourceAttributesGenerated || tt.wantEventsGenerated

			var contents []byte
			if tt.wantMetricsGenerated {
				require.FileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_metrics.go"))
				require.FileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_metrics_test.go"))
				require.FileExists(t, filepath.Join(tmpdir, "documentation.md"))
				if len(tt.wantAttributes) > 0 {
					contents, err = os.ReadFile(filepath.Clean(filepath.Join(tmpdir, "documentation.md")))
					require.NoError(t, err)
					for _, attr := range tt.wantAttributes {
						require.Contains(t, string(contents), attr)
					}
				}
				contents, err = os.ReadFile(filepath.Clean(filepath.Join(tmpdir, generatedPackageDir, "generated_metrics.go")))
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

			if tt.wantLogsGenerated {
				require.FileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_logs.go"))
				require.FileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_logs_test.go"))
			} else {
				require.NoFileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_logs.go"))
				require.NoFileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_logs_test.go"))
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
				contents, err = os.ReadFile(filepath.Clean(filepath.Join(tmpdir, generatedPackageDir, "generated_telemetry.go")))
				require.NoError(t, err)
				if tt.wantMetricsContext {
					require.Contains(t, string(contents), "\"context\"")
				} else {
					require.NotContains(t, string(contents), "\"context\"")
				}
			} else {
				require.NoFileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_telemetry.go"))
			}

			if wantDocumentationGenerated {
				require.FileExists(t, filepath.Join(tmpdir, "documentation.md"))
			} else {
				require.NoFileExists(t, filepath.Join(tmpdir, "documentation.md"))
			}

			if tt.wantStatusGenerated {
				require.FileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_status.go"))
			} else {
				require.NoFileExists(t, filepath.Join(tmpdir, generatedPackageDir, "generated_status.go"))
			}

			contents, err = os.ReadFile(filepath.Clean(filepath.Join(tmpdir, "README.md")))
			require.NoError(t, err)
			if tt.wantReadmeGenerated {
				require.NotContains(t, string(contents), "foo")
			} else {
				require.Contains(t, string(contents), "foo")
			}

			if tt.wantComponentTestGenerated {
				require.FileExists(t, filepath.Join(tmpdir, "generated_component_test.go"))
				contents, err = os.ReadFile(filepath.Clean(filepath.Join(tmpdir, "generated_component_test.go")))
				require.NoError(t, err)
				require.Contains(t, string(contents), "func Test")
				_, err = parser.ParseFile(token.NewFileSet(), "", contents, parser.DeclarationErrors)
				require.NoError(t, err)
			} else {
				require.NoFileExists(t, filepath.Join(tmpdir, "generated_component_test.go"))
			}

			require.FileExists(t, filepath.Join(tmpdir, "generated_package_test.go"))
			contents, err = os.ReadFile(filepath.Clean(filepath.Join(tmpdir, "generated_package_test.go")))
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

			if tt.wantConfigSchemaGenerated {
				require.FileExists(t, filepath.Join(tmpdir, "config.schema.json"))
			} else {
				require.NoFileExists(t, filepath.Join(tmpdir, "config.schema.json"))
			}

			schemaYamlPath := filepath.Join(tmpdir, generatedPackageDir, "config.schema.yaml")
			if tt.wantMetricsSchemaYamlGenerated {
				require.FileExists(t, schemaYamlPath)
				contents, err = os.ReadFile(filepath.Clean(schemaYamlPath))
				require.NoError(t, err)
				require.Contains(t, string(contents), "# Code generated by mdatagen. DO NOT EDIT.")
				require.Contains(t, string(contents), "$defs:")
				if tt.wantMetricsGenerated {
					require.Contains(t, string(contents), "metrics_config:")
					require.Contains(t, string(contents), "metrics_builder_config:")
				}
				if tt.wantEventsGenerated {
					require.Contains(t, string(contents), "events_config:")
					require.Contains(t, string(contents), "logs_builder_config:")
				}
				if tt.wantResourceAttributesGenerated {
					require.Contains(t, string(contents), "resource_attributes_config:")
				}
			} else {
				require.NoFileExists(t, schemaYamlPath)
			}
		})
	}
}

func TestGenerateConfigFiles(t *testing.T) {
	tests := []struct {
		name    string
		md      Metadata
		wantErr bool
		wantGen bool
	}{
		{
			name: "nil config skips generation",
			md: Metadata{
				Type: "test",
				Status: &Status{
					Class: "receiver",
				},
				Config: nil,
			},
			wantGen: false,
		},
		{
			name: "valid config generates schema file",
			md: Metadata{
				Type:        "test",
				PackageName: "shortname",
				Status: &Status{
					Class: "receiver",
				},
				Config: &cfggen.ConfigMetadata{
					Type: "object",
				},
			},
			wantGen: true,
		},
		{
			name: "invalid ref in config causes resolve error",
			md: Metadata{
				Type:        "test",
				PackageName: "shortname",
				Status: &Status{
					Class: "receiver",
				},
				// A local ref without a definition name fails Validate() inside ResolveSchema
				Config: &cfggen.ConfigMetadata{
					Ref: "/config/configauth",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()

			tmpdir := filepath.Join(root, "shortname")
			require.NoError(t, os.MkdirAll(tmpdir, 0o700))

			require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testmodule\n"), 0o600))
			err := generateConfigFiles(tt.md, tmpdir, "testmodule")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantGen {
				require.FileExists(t, filepath.Join(tmpdir, "config.schema.json"))
			} else {
				require.NoFileExists(t, filepath.Join(tmpdir, "config.schema.json"))
			}
		})
	}
}

func TestGenerateConfigGoStruct_RootPackageError(t *testing.T) {
	// tmpdir has no go.mod in any ancestor, so helpers.RootPackage fails
	md := Metadata{
		Type:        "test",
		PackageName: "shortname",
		Status:      &Status{Class: "receiver"},
		Config:      &cfggen.ConfigMetadata{Type: "object"},
	}
	err := generateConfigGoStruct(md, t.TempDir())
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to determine root package")
}

func TestGenerateConfigGoStruct_ResolvedImports(t *testing.T) {
	root := t.TempDir()
	outputDir := filepath.Join(root, "shortname")
	require.NoError(t, os.MkdirAll(outputDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testmodule\n"), 0o600))

	md := Metadata{
		Type:        "test",
		PackageName: "testmodule/shortname",
		Status:      &Status{Class: "receiver"},
		Config: &cfggen.ConfigMetadata{
			Type: "object",
			AllOf: []*cfggen.ConfigMetadata{
				{
					Type:         "object",
					ResolvedFrom: "go.opentelemetry.io/collector/scraper/scraperhelper.ControllerConfig",
					Properties: map[string]*cfggen.ConfigMetadata{
						"timeout": {
							Type:   "string",
							GoType: "time.Duration",
						},
					},
					Default: map[string]any{"timeout": "30s"},
				},
			},
		},
	}

	err := generateConfigGoStruct(md, outputDir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(outputDir, "generated_config.go")) // #nosec G304
	require.NoError(t, err)

	generated := string(content)
	require.Contains(t, generated, `"go.opentelemetry.io/collector/component"`)
	require.Contains(t, generated, `"go.opentelemetry.io/collector/scraper/scraperhelper"`)
	require.Contains(t, generated, "func createDefaultConfig() component.Config")
}

func TestGenerateConfigGoStruct_PropertyDefaultsAndImports(t *testing.T) {
	root := t.TempDir()
	outputDir := filepath.Join(root, "shortname")
	require.NoError(t, os.MkdirAll(outputDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testmodule\n"), 0o600))

	md := Metadata{
		Type:        "test",
		PackageName: "testmodule/shortname",
		Status:      &Status{Class: "receiver"},
		Config: &cfggen.ConfigMetadata{
			Type: "object",
			Properties: map[string]*cfggen.ConfigMetadata{
				"timeout": {
					Type:    "string",
					GoType:  "time.Duration",
					Default: "30s",
				},
			},
		},
	}

	err := generateConfigGoStruct(md, outputDir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(outputDir, "generated_config.go")) // #nosec G304
	require.NoError(t, err)

	generated := string(content)
	require.Contains(t, generated, `"time"`)
	require.Contains(t, generated, "func createDefaultConfig() component.Config")
	require.Contains(t, generated, "Timeout: 30 * time.Second,")
}

func TestGenerateConfigGoStruct_InternalResolvedRefGeneratesLocalType(t *testing.T) {
	root := t.TempDir()
	outputDir := filepath.Join(root, "shortname")
	require.NoError(t, os.MkdirAll(outputDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testmodule\n"), 0o600))

	md := Metadata{
		Type:        "test",
		PackageName: "testmodule/shortname",
		Status:      &Status{Class: "receiver"},
		Config: &cfggen.ConfigMetadata{
			Type: "object",
			Properties: map[string]*cfggen.ConfigMetadata{
				"config": {
					Type:         "object",
					ResolvedFrom: "plain_config",
					Default:      map[string]any{"timeout": "30s"},
					Properties: map[string]*cfggen.ConfigMetadata{
						"timeout": {
							Type:    "string",
							GoType:  "time.Duration",
							Default: "30s",
						},
					},
				},
			},
		},
	}

	err := generateConfigGoStruct(md, outputDir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(outputDir, "generated_config.go")) // #nosec G304
	require.NoError(t, err)

	generated := string(content)
	require.Contains(t, generated, `"time"`)
	require.Contains(t, generated, "type PlainConfig struct")
	require.Contains(t, generated, "func NewDefaultPlainConfig() PlainConfig")
	require.Contains(t, generated, "Timeout: 30 * time.Second,")
	require.Contains(t, generated, "Config PlainConfig `mapstructure:\"config\"`")
	require.Contains(t, generated, "config := NewDefaultPlainConfig()")
	require.Contains(t, generated, "Config: config,")
}

func TestGenerateConfigFiles_GoStructError(t *testing.T) {
	// generateConfigGoStruct fails because tmpdir has no go.mod in any ancestor
	md := Metadata{
		Type:        "test",
		PackageName: "shortname",
		Status:      &Status{Class: "receiver"},
		Config:      &cfggen.ConfigMetadata{Type: "object"},
	}
	err := generateConfigFiles(md, t.TempDir(), "testmodule")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to generate config Go struct")
}

func TestGenerateConfigFiles_WriteError(t *testing.T) {
	md := Metadata{
		Type:        "test",
		PackageName: "shortname",
		Status: &Status{
			Class: "receiver",
		},
		Config: &cfggen.ConfigMetadata{
			Type: "object",
		},
	}
	err := generateConfigFiles(md, "/nonexistent/path/that/does/not/exist", "testmodule")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to write config schema")
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
			err := run(tt.args.ymlPath)
			if !tt.wantErr {
				require.NoError(t, err, "run()")
			} else {
				require.Error(t, err)
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
		deprecation    map[string]DeprecationInfo
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
			name: "readme with status for converter",
			markdown: `# Some component

<!-- status autogenerated section -->
<!-- end autogenerated section -->

Some info about a component
`,
			outputFile:     "readme_with_status_converter.md",
			componentClass: "converter",
			distros:        []string{"contrib"},
		},
		{
			name: "readme with status for provider",
			markdown: `# Some component

<!-- status autogenerated section -->
<!-- end autogenerated section -->

Some info about a component
`,
			outputFile:     "readme_with_status_provider.md",
			componentClass: "provider",
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
				Active: []string{"open-telemetry/collector-approvers"},
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
			name: "readme with multiple signals and deprecation",
			markdown: `# Some component

<!-- status autogenerated section -->
<!-- end autogenerated section -->

Some info about a component
`,
			outputFile: "readme_with_multiple_signals_and_deprecation.md",
			stability: map[component.StabilityLevel][]string{
				component.StabilityLevelBeta:       {"metrics"},
				component.StabilityLevelAlpha:      {"logs"},
				component.StabilityLevelDeprecated: {"traces"},
			},
			deprecation: DeprecationMap{
				"traces": DeprecationInfo{
					Date:      "2025-02-05",
					Migration: "no migration needed",
				},
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
					DisableCodeCov: true,
					Stability:      stability,
					Distributions:  tt.distros,
					Class:          tt.componentClass,
					Warnings:       tt.warnings,
					Codeowners:     tt.codeowners,
					Deprecation:    tt.deprecation,
				},
			}
			tmpdir := t.TempDir()

			readmeFile := filepath.Join(tmpdir, "README.md")
			require.NoError(t, os.WriteFile(readmeFile, []byte(tt.markdown), 0o600))

			err := inlineReplace("templates/readme.md.tmpl", readmeFile, md, statusStart, statusEnd, "metadata", "go.opentelemetry.io/collector")
			require.NoError(t, err)

			require.FileExists(t, filepath.Join(tmpdir, "README.md"))
			got, err := os.ReadFile(filepath.Clean(filepath.Join(tmpdir, "README.md")))
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

// Package metadata contains the autogenerated telemetry and
// build information for the receiver/foo component.
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

// Package metadata contains the autogenerated telemetry and
// build information for the receiver/foo component.
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
		{
			name: "foo component with deprecated type",
			md: Metadata{
				Type:           "foo",
				DeprecatedType: "old_foo",
				Status: &Status{
					Stability: map[component.StabilityLevel][]string{
						component.StabilityLevelBeta: {"metrics"},
					},
					Distributions: []string{"contrib"},
					Class:         "receiver",
				},
			},
			expected: `// Code generated by mdatagen. DO NOT EDIT.

// Package metadata contains the autogenerated telemetry and
// build information for the receiver/foo component.
package metadata

import (
	"go.opentelemetry.io/collector/component"
)

var (
	Type           = component.MustNewType("foo")
	DeprecatedType = component.MustNewType("old_foo")
	ScopeName      = ""
)

const (
	MetricsStability = component.StabilityLevelBeta
)
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			err := generateFile("templates/status.go.tmpl",
				filepath.Join(tmpdir, "generated_status.go"), tt.md, "metadata", "go.opentelemetry.io/collector")
			require.NoError(t, err)
			actual, err := os.ReadFile(filepath.Clean(filepath.Join(tmpdir, "generated_status.go")))
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
)

func Meter(settings component.TelemetrySettings) metric.Meter {
	return settings.MeterProvider.Meter("")
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
)

func Meter(settings component.TelemetrySettings) metric.Meter {
	return settings.MeterProvider.Meter("")
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
				filepath.Join(tmpdir, "generated_telemetry.go"), tt.md, "metadata", "go.opentelemetry.io/collector")
			require.NoError(t, err)
			actual, err := os.ReadFile(filepath.Clean(filepath.Join(tmpdir, "generated_telemetry.go")))
			require.NoError(t, err)
			require.Equal(t, tt.expected, string(actual))
		})
	}
}

func TestGenerateConfigGoStruct_GeneratesTestFile(t *testing.T) {
	root := t.TempDir()
	outputDir := filepath.Join(root, "shortname")
	require.NoError(t, os.MkdirAll(outputDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testmodule\n"), 0o600))

	md := Metadata{
		Type:        "test",
		PackageName: "testmodule/shortname",
		Status:      &Status{Class: "receiver"},
		Config:      &cfggen.ConfigMetadata{Type: "object"},
	}

	require.NoError(t, generateConfigGoStruct(md, outputDir))

	require.FileExists(t, filepath.Join(outputDir, "generated_config.go"))
	require.FileExists(t, filepath.Join(outputDir, "generated_config_test.go"))

	content, err := os.ReadFile(filepath.Join(outputDir, "generated_config_test.go")) // #nosec G304
	require.NoError(t, err)
	require.Contains(t, string(content), "func TestCreateDefaultConfig(")
}

func TestGenerateConfigGoStruct_TestFileContainsValidateTestWhenValidatorsPresent(t *testing.T) {
	root := t.TempDir()
	outputDir := filepath.Join(root, "shortname")
	require.NoError(t, os.MkdirAll(outputDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testmodule\n"), 0o600))

	md := Metadata{
		Type:        "test",
		PackageName: "testmodule/shortname",
		Status:      &Status{Class: "receiver"},
		Config: &cfggen.ConfigMetadata{
			Type: "object",
			Properties: map[string]*cfggen.ConfigMetadata{
				"name": {
					Type:      "string",
					MinLength: func() *int { v := 1; return &v }(),
				},
			},
		},
	}

	require.NoError(t, generateConfigGoStruct(md, outputDir))

	content, err := os.ReadFile(filepath.Join(outputDir, "generated_config_test.go")) // #nosec G304
	require.NoError(t, err)
	require.Contains(t, string(content), "func TestCreateDefaultConfig(")
	require.Contains(t, string(content), "func TestConfigValidate_DefaultValid(")
}

func TestGenerateConfigGoStruct_TestFileNoValidateTestWhenNoValidators(t *testing.T) {
	root := t.TempDir()
	outputDir := filepath.Join(root, "shortname")
	require.NoError(t, os.MkdirAll(outputDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testmodule\n"), 0o600))

	md := Metadata{
		Type:        "test",
		PackageName: "testmodule/shortname",
		Status:      &Status{Class: "receiver"},
		Config: &cfggen.ConfigMetadata{
			Type: "object",
			Properties: map[string]*cfggen.ConfigMetadata{
				"timeout": {
					Type:   "string",
					GoType: "time.Duration",
				},
			},
		},
	}

	require.NoError(t, generateConfigGoStruct(md, outputDir))

	content, err := os.ReadFile(filepath.Join(outputDir, "generated_config_test.go")) // #nosec G304
	require.NoError(t, err)
	require.Contains(t, string(content), "func TestCreateDefaultConfig(")
	require.NotContains(t, string(content), "func TestConfigValidate_DefaultValid(")
}

func TestGenerateConfigGoStruct_BothFileErrorsAccumulated(t *testing.T) {
	root := t.TempDir()
	// outputDir itself does not exist — both generateFileWithFns calls will fail
	outputDir := filepath.Join(root, "nonexistent", "shortname")
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testmodule\n"), 0o600))

	md := Metadata{
		Type:        "test",
		PackageName: "testmodule/nonexistent/shortname",
		Status:      &Status{Class: "receiver"},
		Config:      &cfggen.ConfigMetadata{Type: "object"},
	}

	err := generateConfigGoStruct(md, outputDir)
	require.Error(t, err)
	// Both generated_config.go and generated_config_test.go writes fail; both errors must be present.
	require.Contains(t, err.Error(), "generated_config.go")
	require.Contains(t, err.Error(), "generated_config_test.go")
}

func TestGenerateConfigSchema_LocalizesSameRootRefs(t *testing.T) {
	enabled := true
	md := Metadata{
		Type: "foo",
		ResourceAttributes: map[AttributeName]Attribute{
			"resource.attr": {
				Description: "resource attr",
				EnabledPtr:  &enabled,
				FullName:    "resource.attr",
			},
		},
		Events: map[EventName]Event{
			"default.event": {
				Signal: Signal{
					Enabled:     true,
					Description: "event description",
				},
			},
		},
	}

	tmpdir := t.TempDir()
	outputFile := filepath.Join(tmpdir, "config.schema.yaml")
	err := generateFile("templates/config.schema.yaml.tmpl", outputFile, md, "metadata", "go.opentelemetry.io/collector")
	require.NoError(t, err)

	actual, err := os.ReadFile(filepath.Clean(outputFile))
	require.NoError(t, err)
	require.Contains(t, string(actual), "$ref: /filter.config")
	require.NotContains(t, string(actual), "$ref: go.opentelemetry.io/collector/filter.config")
}
