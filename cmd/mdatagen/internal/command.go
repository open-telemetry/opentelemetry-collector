// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	statusStart = "<!-- status autogenerated section -->"
	statusEnd   = "<!-- end autogenerated section -->"
)

func getVersion() (string, error) {
	// the second returned value is a boolean, which is true if the binaries are built with module support.
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", errors.New("could not read build info")
	}
	return info.Main.Version, nil
}

// NewCommand constructs a new cobra.Command using the given Settings.
// Any URIs specified in CollectorSettings.ConfigProviderSettings.ResolverSettings.URIs
// are considered defaults and will be overwritten by config flags passed as
// command-line arguments to the executable.
// At least one Provider must be set.
func NewCommand() (*cobra.Command, error) {
	ver, err := getVersion()
	if err != nil {
		return nil, err
	}
	rootCmd := &cobra.Command{
		Use:          "mdatagen",
		Version:      ver,
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, args []string) error {
			return run(args[0])
		},
	}
	return rootCmd, nil
}

func run(ymlPath string) error {
	if ymlPath == "" {
		return errors.New("argument must be metadata.yaml file")
	}
	ymlPath, err := filepath.Abs(ymlPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %v: %w", ymlPath, err)
	}

	ymlDir := filepath.Dir(ymlPath)
	packageName := filepath.Base(ymlDir)

	md, err := LoadMetadata(ymlPath)
	if err != nil {
		return fmt.Errorf("failed loading %v: %w", ymlPath, err)
	}

	tmplDir := "templates"

	codeDir := filepath.Join(ymlDir, "internal", md.GeneratedPackageName)
	if err = os.MkdirAll(codeDir, 0o700); err != nil {
		return fmt.Errorf("unable to create output directory %q: %w", codeDir, err)
	}
	testDir := filepath.Join(ymlDir, "internal", md.GeneratedPackageName+"test")
	if err = os.MkdirAll(testDir, 0o700); err != nil {
		return fmt.Errorf("unable to create output test directory %q: %w", codeDir, err)
	}
	if md.Status != nil {
		if md.Status.Class != "cmd" && md.Status.Class != "pkg" && !md.Status.NotComponent {
			if err = generateFile(filepath.Join(tmplDir, "status.go.tmpl"),
				filepath.Join(codeDir, "generated_status.go"), md, "metadata"); err != nil {
				return err
			}
			if err = generateFile(filepath.Join(tmplDir, "component_test.go.tmpl"),
				filepath.Join(ymlDir, "generated_component_test.go"), md, packageName); err != nil {
				return err
			}
		}

		if err = generateFile(filepath.Join(tmplDir, "package_test.go.tmpl"),
			filepath.Join(ymlDir, "generated_package_test.go"), md, packageName); err != nil {
			return err
		}

		if _, err = os.Stat(filepath.Join(ymlDir, "README.md")); err == nil {
			if err = inlineReplace(
				filepath.Join(tmplDir, "readme.md.tmpl"),
				filepath.Join(ymlDir, "README.md"),
				md, statusStart, statusEnd, md.GeneratedPackageName); err != nil {
				return err
			}
		}
	}

	// TODO: Remove this after version v0.122.0 when all the deprecated code should be deleted.
	//  https://github.com/open-telemetry/opentelemetry-collector/issues/12067
	if err = os.Remove(filepath.Join(ymlDir, "generated_component_telemetry_test.go")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("unable to remove generated file \"generated_component_telemetry_test.go\": %w", err)
	}

	toGenerate := map[string]string{}
	if len(md.Telemetry.Metrics) != 0 { // if there are telemetry metrics, generate telemetry specific files
		toGenerate[filepath.Join(tmplDir, "telemetry.go.tmpl")] = filepath.Join(codeDir, "generated_telemetry.go")
		toGenerate[filepath.Join(tmplDir, "telemetry_test.go.tmpl")] = filepath.Join(codeDir, "generated_telemetry_test.go")
		toGenerate[filepath.Join(tmplDir, "telemetrytest.go.tmpl")] = filepath.Join(testDir, "generated_telemetrytest.go")
		toGenerate[filepath.Join(tmplDir, "telemetrytest_test.go.tmpl")] = filepath.Join(testDir, "generated_telemetrytest_test.go")
	}

	if len(md.Metrics) != 0 || len(md.Telemetry.Metrics) != 0 || len(md.ResourceAttributes) != 0 { // if there's metrics or internal metrics, generate documentation for them
		toGenerate[filepath.Join(tmplDir, "documentation.md.tmpl")] = filepath.Join(ymlDir, "documentation.md")
	}

	for tmpl, dst := range toGenerate {
		if err = generateFile(tmpl, dst, md, "metadata"); err != nil {
			return err
		}
	}

	if len(md.Metrics) == 0 && len(md.ResourceAttributes) == 0 {
		return nil
	}

	if err = os.MkdirAll(filepath.Join(codeDir, "testdata"), 0o700); err != nil {
		return fmt.Errorf("unable to create output directory %q: %w", filepath.Join(codeDir, "testdata"), err)
	}

	toGenerate = map[string]string{
		filepath.Join(tmplDir, "testdata", "config.yaml.tmpl"): filepath.Join(codeDir, "testdata", "config.yaml"),
		filepath.Join(tmplDir, "config.go.tmpl"):               filepath.Join(codeDir, "generated_config.go"),
		filepath.Join(tmplDir, "config_test.go.tmpl"):          filepath.Join(codeDir, "generated_config_test.go"),
	}

	if len(md.ResourceAttributes) > 0 { // only generate resource files if resource attributes are configured
		toGenerate[filepath.Join(tmplDir, "resource.go.tmpl")] = filepath.Join(codeDir, "generated_resource.go")
		toGenerate[filepath.Join(tmplDir, "resource_test.go.tmpl")] = filepath.Join(codeDir, "generated_resource_test.go")
	}

	if len(md.Metrics) > 0 { // only generate metrics if metrics are present
		toGenerate[filepath.Join(tmplDir, "metrics.go.tmpl")] = filepath.Join(codeDir, "generated_metrics.go")
		toGenerate[filepath.Join(tmplDir, "metrics_test.go.tmpl")] = filepath.Join(codeDir, "generated_metrics_test.go")
	}

	for tmpl, dst := range toGenerate {
		if err = generateFile(tmpl, dst, md, md.GeneratedPackageName); err != nil {
			return err
		}
	}

	return nil
}

func templatize(tmplFile string, md Metadata) *template.Template {
	return template.Must(
		template.
			New(filepath.Base(tmplFile)).
			Option("missingkey=error").
			Funcs(map[string]any{
				"publicVar": func(s string) (string, error) {
					return FormatIdentifier(s, true)
				},
				"attributeInfo": func(an AttributeName) Attribute {
					return md.Attributes[an]
				},
				"metricInfo": func(mn MetricName) Metric {
					return md.Metrics[mn]
				},
				"telemetryInfo": func(mn MetricName) Metric {
					return md.Telemetry.Metrics[mn]
				},
				"parseImportsRequired": func(metrics map[MetricName]Metric) bool {
					for _, m := range metrics {
						if m.Data().HasMetricInputType() {
							return true
						}
					}
					return false
				},
				"stringsJoin":  strings.Join,
				"stringsSplit": strings.Split,
				"userLinks": func(elems []string) []string {
					result := make([]string, len(elems))
					for i, elem := range elems {
						if elem == "open-telemetry/collector-approvers" {
							result[i] = "[@open-telemetry/collector-approvers](https://github.com/orgs/open-telemetry/teams/collector-approvers)"
						} else {
							result[i] = fmt.Sprintf("[@%s](https://www.github.com/%s)", elem, elem)
						}
					}
					return result
				},
				"casesTitle":  cases.Title(language.English).String,
				"toLowerCase": strings.ToLower,
				"toCamelCase": func(s string) string {
					caser := cases.Title(language.English).String
					parts := strings.Split(s, "_")
					result := ""
					for _, part := range parts {
						result += caser(part)
					}
					return result
				},
				"inc":       func(i int) int { return i + 1 },
				"distroURL": distroURL,
				"isExporter": func() bool {
					return md.Status.Class == "exporter"
				},
				"isProcessor": func() bool {
					return md.Status.Class == "processor"
				},
				"isReceiver": func() bool {
					return md.Status.Class == "receiver"
				},
				"isExtension": func() bool {
					return md.Status.Class == "extension"
				},
				"isConnector": func() bool {
					return md.Status.Class == "connector"
				},
				"isScraper": func() bool {
					return md.Status.Class == "scraper"
				},
				"isCommand": func() bool {
					return md.Status.Class == "cmd"
				},
				"supportsLogs": func() bool {
					for _, signals := range md.Status.Stability {
						for _, s := range signals {
							if s == "logs" {
								return true
							}
						}
					}
					return false
				},
				"supportsMetrics": func() bool {
					for _, signals := range md.Status.Stability {
						for _, s := range signals {
							if s == "metrics" {
								return true
							}
						}
					}
					return false
				},
				"supportsTraces": func() bool {
					for _, signals := range md.Status.Stability {
						for _, s := range signals {
							if s == "traces" {
								return true
							}
						}
					}
					return false
				},
				"supportsLogsToLogs": func() bool {
					for _, signals := range md.Status.Stability {
						for _, s := range signals {
							if s == "logs_to_logs" {
								return true
							}
						}
					}
					return false
				},
				"supportsLogsToMetrics": func() bool {
					for _, signals := range md.Status.Stability {
						for _, s := range signals {
							if s == "logs_to_metrics" {
								return true
							}
						}
					}
					return false
				},
				"supportsLogsToTraces": func() bool {
					for _, signals := range md.Status.Stability {
						for _, s := range signals {
							if s == "logs_to_traces" {
								return true
							}
						}
					}
					return false
				},
				"supportsMetricsToLogs": func() bool {
					for _, signals := range md.Status.Stability {
						for _, s := range signals {
							if s == "metrics_to_logs" {
								return true
							}
						}
					}
					return false
				},
				"supportsMetricsToMetrics": func() bool {
					for _, signals := range md.Status.Stability {
						for _, s := range signals {
							if s == "metrics_to_metrics" {
								return true
							}
						}
					}
					return false
				},
				"supportsMetricsToTraces": func() bool {
					for _, signals := range md.Status.Stability {
						for _, s := range signals {
							if s == "metrics_to_traces" {
								return true
							}
						}
					}
					return false
				},
				"supportsTracesToLogs": func() bool {
					for _, signals := range md.Status.Stability {
						for _, s := range signals {
							if s == "traces_to_logs" {
								return true
							}
						}
					}
					return false
				},
				"supportsTracesToMetrics": func() bool {
					for _, signals := range md.Status.Stability {
						for _, s := range signals {
							if s == "traces_to_metrics" {
								return true
							}
						}
					}
					return false
				},
				"supportsTracesToTraces": func() bool {
					for _, signals := range md.Status.Stability {
						for _, s := range signals {
							if s == "traces_to_traces" {
								return true
							}
						}
					}
					return false
				},
				"expectConsumerError": func() bool {
					return md.Tests.ExpectConsumerError
				},
				// ParseFS delegates the parsing of the files to `Glob`
				// which uses the `\` as a special character.
				// Meaning on windows based machines, the `\` needs to be replaced
				// with a `/` for it to find the file.
			}).ParseFS(TemplateFS, strings.ReplaceAll(tmplFile, "\\", "/")))
}

func executeTemplate(tmplFile string, md Metadata, goPackage string) ([]byte, error) {
	tmpl := templatize(tmplFile, md)
	buf := bytes.Buffer{}

	if err := tmpl.Execute(&buf, TemplateContext{Metadata: md, Package: goPackage}); err != nil {
		return []byte{}, fmt.Errorf("failed executing template: %w", err)
	}
	return buf.Bytes(), nil
}

func inlineReplace(tmplFile string, outputFile string, md Metadata, start string, end string, goPackage string) error {
	var readmeContents []byte
	var err error
	if readmeContents, err = os.ReadFile(outputFile); err != nil { //nolint:gosec
		return err
	}

	re := regexp.MustCompile(fmt.Sprintf("%s[\\s\\S]*%s", start, end))
	if !re.Match(readmeContents) {
		return nil
	}

	if md.GithubProject == "" {
		md.GithubProject = "open-telemetry/opentelemetry-collector-contrib"
	}

	buf, err := executeTemplate(tmplFile, md, goPackage)
	if err != nil {
		return err
	}

	s := re.ReplaceAllString(string(readmeContents), string(buf))
	if err := os.WriteFile(outputFile, []byte(s), 0o600); err != nil {
		return fmt.Errorf("failed writing %q: %w", outputFile, err)
	}

	return nil
}

func generateFile(tmplFile string, outputFile string, md Metadata, goPackage string) error {
	if err := os.Remove(outputFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("unable to remove generated file %q: %w", outputFile, err)
	}

	result, err := executeTemplate(tmplFile, md, goPackage)
	if err != nil {
		return err
	}
	var formatErr error
	if strings.HasSuffix(outputFile, ".go") {
		if formatted, err := format.Source(result); err == nil {
			result = formatted
		} else {
			formatErr = fmt.Errorf("failed formatting %s:%w", outputFile, err)
		}
	}

	if err := os.WriteFile(outputFile, result, 0o600); err != nil {
		return fmt.Errorf("failed writing %q: %w", outputFile, err)
	}

	return formatErr
}
