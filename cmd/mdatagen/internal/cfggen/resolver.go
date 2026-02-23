// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	schemaVersion = "https://json-schema.org/draft/2020-12/schema"
	// goDurationPattern matches Go duration strings (e.g., "30s", "1h30m", "500ms")
	goDurationPattern = `^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`
)

type Resolver struct {
	pkgID  string
	class  string
	name   string
	loader Loader
}

func NewResolver(pkgID, class, name, dir string) *Resolver {
	loader := NewLoader(dir)

	return &Resolver{
		loader: loader,
		pkgID:  pkgID,
		class:  class,
		name:   name,
	}
}

// ResolveSchema takes a source configuration metadata schema and resolves all references ($ref)
// to produce a fully resolved schema. It handles both internal references (within the same schema) and external references
// (pointing to other schemas, either locally or remotely). The resolver uses registered loaders to fetch external schemas as needed.
//
// Returns a new ConfigMetadata with all references resolved, or an error if resolution fails.
func (r *Resolver) ResolveSchema(src *ConfigMetadata) (*ConfigMetadata, error) {
	target := &ConfigMetadata{}
	err := r.resolveSchema(src, src, target)
	if err != nil {
		return nil, err
	}

	target.Schema = schemaVersion
	target.ID = r.pkgID
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
	ref := NewRef(current.Ref)

	if err := ref.Validate(); err != nil {
		return nil, fmt.Errorf("invalid reference format %q: %w", current.Ref, err)
	}

	if ref.isInternal() {
		if root.Defs != nil {
			if val, ok := root.Defs[ref.DefName()]; ok {
				return val, nil
			}
		}
		return nil, fmt.Errorf("internal reference %q not found in $defs", current.Ref)
	}

	if ref.isExternal() {
		// check if it's in known namespace
		if _, ok := ref.Namespace(); !ok {
			// fallback to type "any"
			current.GoType = current.Ref
			current.Comment = "Empty or unknown reference, defaulting to 'any' type."
			return current, nil
		}
	}

	// attempt to load external reference using registered loaders
	return r.loadExternalRef(ref)
}

// loadExternalRef uses SchemaLoader to load external references
func (r *Resolver) loadExternalRef(ref *Ref) (*ConfigMetadata, error) {
	md, err := r.loader.Load(*ref)
	if err != nil {
		return nil, err
	}
	if md == nil {
		return nil, fmt.Errorf("no loader could resolve external reference: %s", ref)
	}

	resolved := &ConfigMetadata{}
	if err := r.resolveSchema(md, md, resolved); err != nil {
		return nil, fmt.Errorf("failed to resolve internal references in external schema %s: %w", ref, err)
	}

	if resolved.Defs != nil {
		if def, ok := resolved.Defs[ref.DefName()]; ok {
			return def, nil
		}
	}

	return nil, fmt.Errorf("type %q not found in loaded schema for reference %s", ref.DefName(), ref)
}
