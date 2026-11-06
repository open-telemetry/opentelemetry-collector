// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/builder/internal"

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/cmd/builder/internal/builder"
	"go.opentelemetry.io/collector/cmd/builder/internal/fromconfig"
)

const defaultDescription = "Custom OpenTelemetry Collector"

//go:embed init/templates/*.tmpl
var templatesFS embed.FS

type metadata struct {
	Name          string
	Description   string
	StableVersion string
	BetaVersion   string
}

// initOptions holds the optional settings of the init command.
type initOptions struct {
	// fromConfig is the path to a Collector configuration to build the manifest from.
	fromConfig string
	// resolver resolves the components of fromConfig. Defaults to using the Go module proxy.
	resolver *fromconfig.Resolver
	// stderr receives the progress and warnings of fromConfig. Defaults to io.Discard.
	stderr io.Writer
}

func initCommand() *cobra.Command {
	var outputPath, fromConfig string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "[EXPERIMENTAL] Initializes a new custom collector repository in the provided folder",
		Long:  `ocb init initializes a new repository in the provided folder with a manifest to start building a custom Collector. This command is experimental and very likely to change.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context(), outputPath, initOptions{fromConfig: fromConfig, stderr: cmd.ErrOrStderr()})
		},
	}

	cmd.Flags().StringVar(&outputPath, "path", ".", "Output path where the collector repository will be initialized")
	cmd.Flags().StringVar(&fromConfig, "from-config", "", "Collector configuration to build the manifest from. Looks up the modules of its components on the Go module proxy")

	return cmd
}

// run initializes the repository in outputPath.
func run(ctx context.Context, outputPath string, opts initOptions) error {
	if outputPath == "" {
		return errors.New("argument must be a folder")
	}
	path, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %v: %w", outputPath, err)
	}

	// Resolve the components before writing anything, so that a failure leaves no files behind.
	var source *configSource
	if opts.fromConfig != "" {
		source, err = resolveConfig(ctx, opts)
		if err != nil {
			return err
		}
	}

	err = os.MkdirAll(path, 0o750)
	if err != nil {
		return fmt.Errorf("failed creating folder %v: %w", path, err)
	}

	meta := metadata{
		Name:          filepath.Base(path),
		Description:   defaultDescription,
		StableVersion: builder.DefaultStableOtelColVersion,
		BetaVersion:   builder.DefaultBetaOtelColVersion,
	}

	err = buildManifest(path, meta, source)
	if err != nil {
		return fmt.Errorf("failed building manifest: %w", err)
	}

	err = writeTemplate(path, ".gitignore", meta)
	if err != nil {
		return fmt.Errorf("failed writing gitignore: %w", err)
	}

	err = writeTemplate(path, "README.md", meta)
	if err != nil {
		return fmt.Errorf("failed writing README: %w", err)
	}

	err = writeTemplate(path, "go.mod", meta)
	if err != nil {
		return fmt.Errorf("failed writing go.mod: %w", err)
	}

	err = writeTemplate(path, "Makefile", meta)
	if err != nil {
		return fmt.Errorf("failed writing Makefile: %w", err)
	}

	if source != nil {
		err = os.WriteFile(filepath.Join(path, "config.yaml"), source.data, 0o600) //nolint:gosec // G703: path is the output folder chosen by the user
	} else {
		err = writeTemplate(path, "config.yaml", meta)
	}
	if err != nil {
		return fmt.Errorf("failed writing config.yaml: %w", err)
	}

	err = os.MkdirAll(filepath.Join(path, "build"), 0o750)
	if err != nil {
		return fmt.Errorf("failed creating build folder: %w", err)
	}

	err = runTidy(path)
	if err != nil {
		return fmt.Errorf("failed running go mod tidy: %w", err)
	}

	return nil
}

// configSource is a Collector configuration the manifest is built from.
type configSource struct {
	// name is the path to the configuration, as provided by the user.
	name   string
	data   []byte
	result fromconfig.Result
}

// resolveConfig reads the configuration and resolves the modules of its components.
func resolveConfig(ctx context.Context, opts initOptions) (*configSource, error) {
	data, err := os.ReadFile(opts.fromConfig)
	if err != nil {
		return nil, fmt.Errorf("failed reading configuration: %w", err)
	}

	resolver := opts.resolver
	if resolver == nil {
		resolver = &fromconfig.Resolver{Release: builder.DefaultBetaOtelColVersion}
	}
	result, err := resolver.Resolve(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed resolving components of %v: %w", opts.fromConfig, err)
	}

	stderr := opts.stderr
	if stderr == nil {
		stderr = io.Discard
	}
	for _, r := range result.Resolved {
		fmt.Fprintf(stderr, "ocb: resolved %s %s -> %s\n", r.Class, r.Type, r.GoMod())
	}
	for _, u := range result.Unresolved {
		fmt.Fprintf(stderr, "ocb: warning: %s; add its gomod to manifest.yaml manually\n", u.Reason)
	}
	fmt.Fprintf(stderr, "ocb: resolved %d of %d components referenced by %s\n",
		len(result.Resolved), len(result.Resolved)+len(result.Unresolved), opts.fromConfig)

	return &configSource{name: opts.fromConfig, data: data, result: result}, nil
}

// buildManifest writes the manifest. Without a source, it contains the OTLP receiver and exporter.
func buildManifest(path string, meta metadata, source *configSource) error {
	cfg := builder.Config{
		Distribution: builder.Distribution{
			Name:        meta.Name,
			Description: meta.Description,
			OutputPath:  "./build/collector",
		},
	}

	if source == nil {
		cfg.Exporters = []builder.Module{
			{GoMod: "go.opentelemetry.io/collector/exporter/otlpexporter " + meta.BetaVersion},
		}
		cfg.Receivers = []builder.Module{
			{GoMod: "go.opentelemetry.io/collector/receiver/otlpreceiver " + meta.BetaVersion},
		}
	} else {
		addResolved(&cfg, source.result.Resolved)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed marshaling manifest: %w", err)
	}
	if source != nil {
		data = append(data, unresolvedPlaceholders(source)...)
	}

	return os.WriteFile(filepath.Join(path, "manifest.yaml"), data, 0o600)
}

// addResolved adds the resolved components to the sections of the manifest.
func addResolved(cfg *builder.Config, resolved []fromconfig.Resolved) {
	for _, r := range resolved {
		module := builder.Module{GoMod: r.GoMod()}
		switch r.Class {
		case fromconfig.Receiver:
			cfg.Receivers = append(cfg.Receivers, module)
		case fromconfig.Processor:
			cfg.Processors = append(cfg.Processors, module)
		case fromconfig.Exporter:
			cfg.Exporters = append(cfg.Exporters, module)
		case fromconfig.Extension:
			cfg.Extensions = append(cfg.Extensions, module)
		case fromconfig.Connector:
			cfg.Connectors = append(cfg.Connectors, module)
		}
	}
}

// manifestClasses lists the component classes in the order of the manifest sections.
var manifestClasses = []fromconfig.Class{
	fromconfig.Exporter,
	fromconfig.Extension,
	fromconfig.Receiver,
	fromconfig.Processor,
	fromconfig.Connector,
}

// unresolvedPlaceholders returns a commented-out block listing the components
// that could not be matched to a module, or nothing if all were.
func unresolvedPlaceholders(source *configSource) []byte {
	if len(source.result.Unresolved) == 0 {
		return nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, `
# The following components are referenced by %s but could
# not be matched to a Go module in the opentelemetry-collector or
# opentelemetry-collector-contrib repositories. Fill in the module path and
# version for each one and uncomment it, or remove it from your Collector
# configuration.
#
`, source.name)

	for _, class := range manifestClasses {
		first := true
		for _, u := range source.result.Unresolved {
			if u.Class != class {
				continue
			}
			if first {
				fmt.Fprintf(&sb, "# %ss:\n", class)
				first = false
			}
			fmt.Fprintf(&sb, "#     - gomod: <module path> <version> # %s\n", u.Type)
		}
	}
	return []byte(sb.String())
}

func writeTemplate(path, fn string, m metadata) error {
	outputFile := filepath.Join(path, fn)

	content, err := executeTemplate(fn+".tmpl", m)
	if err != nil {
		return err
	}
	return os.WriteFile(outputFile, content, 0o600)
}

func executeTemplate(tmplFile string, m metadata) ([]byte, error) {
	tmplPath := path.Join("init/templates", tmplFile)
	tmpl := template.Must(template.New(tmplFile).ParseFS(templatesFS, tmplPath))
	buf := bytes.Buffer{}

	if err := tmpl.Execute(&buf, m); err != nil {
		return []byte{}, fmt.Errorf("failed executing template: %w", err)
	}
	return buf.Bytes(), nil
}

func runTidy(path string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = path
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w (%s)", err, string(output))
	}
	return nil
}
