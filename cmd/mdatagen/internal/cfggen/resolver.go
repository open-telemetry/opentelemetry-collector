// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	schemaVersion = "https://json-schema.org/draft/2020-12/schema"
	// goDurationPattern matches Go duration strings (e.g., "30s", "1h30m", "500ms")
	goDurationPattern = `^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`
)

type Resolver struct {
	pkgID   string
	version string
	class   string
	name    string
	loader  Loader
}

func NewResolver(pkgID, class, name, version string) *Resolver {
	schemasDir := getSchemasDir()
	loader := NewLoader(schemasDir)

	return &Resolver{
		loader:  loader,
		pkgID:   pkgID,
		version: version,
		class:   class,
		name:    name,
	}
}

// getSchemasDir returns the path to .schemas directory at the repo root.
// Returns empty string if repo root cannot be determined.
func getSchemasDir() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	repoRoot := strings.TrimSpace(string(output))
	return filepath.Join(repoRoot, ".schemas")
}

// ResolveSchema takes a source configuration metadata schema and resolves all references ($ref)
// to produce a fully resolved schema. It creates a new ConfigMetadata with proper JSON Schema
// metadata (Schema, ID, Title) based on the component's pkgID, class, name, and version.
//
// The resolution process:
//   - Sets the JSON Schema version to draft 2020-12
//   - Generates a unique schema ID from the pkgID namespace, version, and path
//   - Sets the title to "class/name" (e.g., "receiver/otlp")
//   - Recursively resolves all $ref references within the schema
//   - Deep copies all fields while expanding simplified references
//
// Returns a new ConfigMetadata with all references resolved, or an error if resolution fails.
func (r *Resolver) ResolveSchema(src *ConfigMetadata) (*ConfigMetadata, error) {
	ns, err1 := getNamespace(r.pkgID)
	if err1 != nil {
		return nil, fmt.Errorf("failed to get namespace from pkgID %q: %w", r.pkgID, err1)
	}
	path := strings.TrimPrefix(r.pkgID, ns)

	target := &ConfigMetadata{}
	err2 := r.resolveSchema(src, src, target)

	if err2 != nil {
		return nil, err2
	}

	// Set JSON Schema metadata
	target.Schema = schemaVersion
	target.ID = filepath.Join(ns, path)
	target.Title = fmt.Sprintf("%s/%s", r.class, r.name)

	return target, nil
}

// transformDurationFormat converts JSON Schema format: duration to Go duration pattern.
// JSON Schema duration format expects ISO 8601 (e.g., "PT30S"), but Go uses a different
// format (e.g., "30s", "1h30m"). This function replaces the format with a pattern that
// validates Go duration strings.
func transformDurationFormat(md *ConfigMetadata) {
	if md.Type == "string" && md.Format == "duration" {
		md.Format = ""
		md.Pattern = goDurationPattern
		if md.Description != "" && !strings.Contains(md.Description, "duration") {
			md.Description += " (duration format, e.g., \"30s\", \"1h30m\")"
		}
	}
}

func (r *Resolver) resolveSchema(root, current, target *ConfigMetadata) error {
	if current.Ref != "" {
		resolved, err := r.resolveRef(root, current)
		if err != nil {
			return fmt.Errorf("failed to resolve $ref %q: %w", current.Ref, err)
		}
		current = resolved
	}

	transformDurationFormat(current)

	currRef := reflect.ValueOf(current).Elem()
	targetRef := reflect.ValueOf(target).Elem()

	for i := 0; i < currRef.NumField(); i++ {
		field := currRef.Field(i)
		targetField := targetRef.Field(i)

		if !targetField.CanSet() {
			continue
		}

		switch field.Kind() {
		case reflect.Struct:
			if field.Type() == reflect.TypeFor[*ConfigMetadata]() {
				newMeta := &ConfigMetadata{}
				if err := r.resolveSchema(root, field.Addr().Interface().(*ConfigMetadata), newMeta); err != nil {
					return err
				}
				targetField.Set(reflect.ValueOf(newMeta).Elem())
			}
		case reflect.Ptr:
			if !field.IsNil() && field.Elem().Kind() == reflect.Struct {
				if field.Type() == reflect.TypeFor[*ConfigMetadata]() {
					newMeta := &ConfigMetadata{}
					if err := r.resolveSchema(root, field.Interface().(*ConfigMetadata), newMeta); err != nil {
						return err
					}
					targetField.Set(reflect.ValueOf(newMeta))
				}
			}
		case reflect.Map:
			if field.Type().Elem() == reflect.TypeFor[*ConfigMetadata]() {
				newMap := reflect.MakeMap(field.Type())
				iter := field.MapRange()
				for iter.Next() {
					key := iter.Key()
					value := iter.Value()
					if !value.IsNil() {
						newMeta := &ConfigMetadata{}
						if err := r.resolveSchema(root, value.Interface().(*ConfigMetadata), newMeta); err != nil {
							return err
						}
						newMap.SetMapIndex(key, reflect.ValueOf(newMeta))
					}
				}
				targetField.Set(newMap)
			} else {
				targetField.Set(field)
			}
		case reflect.Slice:
			if field.Type().Elem() == reflect.TypeFor[*ConfigMetadata]() {
				newSlice := reflect.MakeSlice(field.Type(), field.Len(), field.Len())
				for j := 0; j < field.Len(); j++ {
					elem := field.Index(j)
					if !elem.IsNil() {
						newMeta := &ConfigMetadata{}
						if err := r.resolveSchema(root, elem.Interface().(*ConfigMetadata), newMeta); err != nil {
							return err
						}
						newSlice.Index(j).Set(reflect.ValueOf(newMeta))
					}
				}
				targetField.Set(newSlice)
			} else {
				targetField.Set(field)
			}
		default:
			targetField.Set(field)
		}
	}

	return nil
}

// resolveRef resolves a JSON Schema $ref, handling both internal and external references
func (r *Resolver) resolveRef(root, current *ConfigMetadata) (*ConfigMetadata, error) {
	ref := current.Ref
	// Handle external references (./internal/metadata.metrics_builder_config,
	// go.opentelemetry.io/collector/scraper/scraperhelper.controller_config)
	if isExternalRef(ref) {
		// These would use the Resolver.loaders in the future
		return r.loadExternalRef(ref)
	}

	// Handle simplified syntax for internal references
	if root.Defs != nil {
		if val, ok := root.Defs[ref]; ok {
			return val, nil
		}
	}

	// fallback to type "any"
	current.GoType = current.Ref
	current.Comment = "Empty or unknown reference, defaulting to 'any' type."
	return current, nil
}

// loadExternalRef uses registered loaders to load external references
func (r *Resolver) loadExternalRef(refPath string) (*ConfigMetadata, error) {
	ref, err := getRef(refPath)
	if err != nil {
		return nil, fmt.Errorf("invalid reference path %q: %w", refPath, err)
	}

	md, err := r.loader.Load(*ref, r.version)
	if err != nil {
		return nil, err
	}
	if md == nil {
		return nil, fmt.Errorf("no loader could resolve external reference: %s", refPath)
	}

	resolved := &ConfigMetadata{}
	if err := r.resolveSchema(md, md, resolved); err != nil {
		return nil, fmt.Errorf("failed to resolve internal references in external schema %s: %w", refPath, err)
	}

	if resolved.Defs != nil {
		if def, ok := resolved.Defs[ref.Type]; ok {
			return def, nil
		}
	}

	return nil, fmt.Errorf("type %q not found in loaded schema for reference %s", ref.Type, refPath)
}

// isExternalRef checks if a reference is an external reference (URL or relative path)
func isExternalRef(ref string) bool {
	if isInNamespace(ref) {
		return true
	}
	if strings.HasPrefix(ref, "./") {
		return true
	}

	return false
}
