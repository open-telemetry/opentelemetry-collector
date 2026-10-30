// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder // import "go.opentelemetry.io/collector/cmd/builder/internal/builder"

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/internal/schemagen"
)

// errMissingComponentMetadata signals that a component module has no metadata.yaml,
// so its component type cannot be determined and it cannot be described in the schema.
var errMissingComponentMetadata = errors.New("metadata.yaml not found")

// componentSchemaFileName is the only schema artifact accepted for embedding.
// config.schema.yaml / .yml are unresolved bootstrap files with dangling $refs;
// only config.schema.json (written by mdatagen) is fully resolved and self-contained.
const componentSchemaFileName = "config.schema.json"

// embeddedSchemaTemplateParams is the data passed to schemaTemplate.
type embeddedSchemaTemplateParams struct {
	// Schema is the combined JSON schema to embed. Empty renders a nil slice.
	Schema string
}

type componentSchemaRef struct {
	section schemagen.CollectorSection
	module  Module
}

type componentSchemaMetadata struct {
	Type           string `yaml:"type"`
	DeprecatedType string `yaml:"deprecated_type"`
}

func writeEmbeddedSchemaSourceFile(cfg *Config, schema []byte) error {
	return processAndWrite(cfg, schemaTemplate, schemaTemplate.Name(), embeddedSchemaTemplateParams{Schema: string(schema)})
}

func generateEmbeddedSchemaWithDeps(cfg *Config, deps builderDeps) error {
	schema, err := buildEmbeddedSchemaWithDeps(cfg, deps, func(modulePath string) (string, error) {
		stdout, runErr := deps.runGoCommand(cfg, "list", "-m", "-f", "{{.Dir}}", modulePath)
		if runErr != nil {
			return "", runErr
		}

		return strings.TrimSpace(string(stdout)), nil
	})
	if err != nil {
		return err
	}

	if err := deps.writeEmbeddedSchemaSourceFile(cfg, schema); err != nil {
		return fmt.Errorf("failed to write embedded schema source: %w", err)
	}

	return nil
}

func buildEmbeddedSchema(cfg *Config, resolveModuleDir func(string) (string, error)) ([]byte, error) {
	return buildEmbeddedSchemaWithDeps(cfg, defaultBuilderDeps(), resolveModuleDir)
}

func buildEmbeddedSchemaWithDeps(cfg *Config, deps builderDeps, resolveModuleDir func(string) (string, error)) ([]byte, error) {
	parts := schemagen.CollectorSchemaParts{}
	schemaFileCount := 0
	// permissiveSections collects the sections that contain at least one component whose
	// type is unknown (no metadata.yaml). Those sections must accept any component key,
	// otherwise the schema would reject valid configurations for that component.
	permissiveSections := map[schemagen.CollectorSection]bool{}

	for _, ref := range deps.selectedComponentSchemaRefs(cfg) {
		modulePath, _, _ := strings.Cut(ref.module.GoMod, " ")
		moduleDir, err := resolveModuleDir(modulePath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve module directory for %q: %w", modulePath, err)
		}

		componentSchema, hasSchemaFile, err := loadCollectorComponentSchema(moduleDir)
		if errors.Is(err, errMissingComponentMetadata) {
			cfg.Logger.Info("Component has no metadata.yaml, its configuration will not be validated by the embedded schema.",
				zap.String("module", modulePath), zap.String("section", string(ref.section)))
			permissiveSections[ref.section] = true
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to load component schema for %q: %w", modulePath, err)
		}
		if hasSchemaFile {
			schemaFileCount++
		}

		switch ref.section {
		case schemagen.CollectorSectionReceivers:
			parts.Receivers = append(parts.Receivers, componentSchema)
		case schemagen.CollectorSectionProcessors:
			parts.Processors = append(parts.Processors, componentSchema)
		case schemagen.CollectorSectionExporters:
			parts.Exporters = append(parts.Exporters, componentSchema)
		case schemagen.CollectorSectionConnectors:
			parts.Connectors = append(parts.Connectors, componentSchema)
		case schemagen.CollectorSectionExtensions:
			parts.Extensions = append(parts.Extensions, componentSchema)
		default:
			return nil, fmt.Errorf("unsupported collector section %q", ref.section)
		}
	}

	if schemaFileCount == 0 {
		return nil, nil
	}

	combined, err := schemagen.CombineCollectorSchema(parts)
	if err != nil {
		return nil, fmt.Errorf("failed to combine collector schema: %w", err)
	}

	for section := range permissiveSections {
		if sectionSchema := combined.Properties[string(section)]; sectionSchema != nil {
			sectionSchema.AdditionalProperties = nil
		}
	}

	return json.MarshalIndent(combined, "", "  ")
}

func selectedComponentSchemaRefs(cfg *Config) []componentSchemaRef {
	refs := make([]componentSchemaRef, 0, len(cfg.Receivers)+len(cfg.Processors)+len(cfg.Exporters)+len(cfg.Connectors)+len(cfg.Extensions))

	appendRefs := func(section schemagen.CollectorSection, modules []Module) {
		for _, mod := range modules {
			refs = append(refs, componentSchemaRef{
				section: section,
				module:  mod,
			})
		}
	}

	appendRefs(schemagen.CollectorSectionReceivers, cfg.Receivers)
	appendRefs(schemagen.CollectorSectionProcessors, cfg.Processors)
	appendRefs(schemagen.CollectorSectionExporters, cfg.Exporters)
	appendRefs(schemagen.CollectorSectionConnectors, cfg.Connectors)
	appendRefs(schemagen.CollectorSectionExtensions, cfg.Extensions)

	return refs
}

func loadCollectorComponentSchema(moduleDir string) (schemagen.CollectorComponentSchema, bool, error) {
	metadataPath := filepath.Join(moduleDir, "metadata.yaml")
	// #nosec G304 -- moduleDir is resolved from go mod list command output
	metadataBytes, err := os.ReadFile(metadataPath)
	if errors.Is(err, os.ErrNotExist) {
		return schemagen.CollectorComponentSchema{}, false, errMissingComponentMetadata
	}
	if err != nil {
		return schemagen.CollectorComponentSchema{}, false, fmt.Errorf("failed to read metadata.yaml: %w", err)
	}

	var metadata componentSchemaMetadata
	if unmarshalErr := yaml.Unmarshal(metadataBytes, &metadata); unmarshalErr != nil {
		return schemagen.CollectorComponentSchema{}, false, fmt.Errorf("failed to parse metadata.yaml: %w", unmarshalErr)
	}
	if metadata.Type == "" {
		return schemagen.CollectorComponentSchema{}, false, errors.New("metadata.yaml missing component type")
	}

	result := schemagen.CollectorComponentSchema{
		Type:           metadata.Type,
		DeprecatedType: metadata.DeprecatedType,
	}

	schemaBytes, found, err := readComponentSchemaFile(moduleDir)
	if err != nil {
		return schemagen.CollectorComponentSchema{}, false, err
	}
	if !found {
		return result, false, nil
	}

	componentSchema, err := parseComponentConfigSchema(schemaBytes)
	if err != nil {
		return schemagen.CollectorComponentSchema{}, false, fmt.Errorf("failed to parse %s: %w", componentSchemaFileName, err)
	}

	result.Schema = componentSchema
	return result, true, nil
}

func readComponentSchemaFile(moduleDir string) ([]byte, bool, error) {
	schemaPath := filepath.Join(moduleDir, componentSchemaFileName)
	// #nosec G304 -- moduleDir is resolved from go mod list command output
	schemaBytes, err := os.ReadFile(schemaPath)
	if err == nil {
		return schemaBytes, true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	return nil, false, fmt.Errorf("failed to read %s: %w", componentSchemaFileName, err)
}

func parseComponentConfigSchema(schemaBytes []byte) (*schemagen.JSONSchema, error) {
	var schema schemagen.JSONSchema
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}
