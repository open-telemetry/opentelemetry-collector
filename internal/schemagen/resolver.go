// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/internal/schemagen"

import (
	"fmt"
	"log"
	"maps"
	"reflect"
	"slices"
)

const (
	schemaVersion = "https://json-schema.org/draft/2020-12/schema"
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
	err := r.resolveSchema(src, src, target, nil)
	if err != nil {
		return nil, err
	}

	target.Schema = schemaVersion
	target.ID = r.pkgID
	target.Title = fmt.Sprintf("%s/%s", r.class, r.name)

	if len(src.Properties) > 0 {
		target.Type = "object"
	}

	return target, nil
}

func (r *Resolver) resolveSchema(root, current, target *ConfigMetadata, origin *Ref) error {
	if current.Ref != "" {
		// Preserve custom extensions defined on the reference node
		customGoType := current.GoType
		customIsPointer := current.IsPointer
		customIsOptional := current.IsOptional
		customDescription := current.Description
		customDefault := current.Default
		customEnum := current.Enum
		customInternalOnly := current.InternalOnly
		customMaxLength := current.MaxLength
		customMinLength := current.MinLength
		customPattern := current.Pattern
		customMaximum := current.Maximum
		customExclusiveMaximum := current.ExclusiveMaximum
		customMinimum := current.Minimum
		customExclusiveMinimum := current.ExclusiveMinimum

		finalRef := current.Ref
		resolved, err := r.resolveRef(root, current, origin)
		if err != nil {
			return fmt.Errorf("failed to resolve $ref %q: %w", current.Ref, err)
		}

		// Follow alias chains: an internal $defs entry may itself be a $ref (e.g. to an external type).
		// Keep resolving until we reach a concrete schema node.
		// Stop if resolveRef returns the same pointer (unknown-ref fallback sets GoType and returns current).
		for resolved.Ref != "" {
			next, err := r.resolveRef(root, resolved, origin)
			if err != nil {
				return fmt.Errorf("failed to resolve $ref %q: %w", resolved.Ref, err)
			}
			if next == resolved {
				// fallback "any" case: resolveRef returned the node unchanged
				break
			}
			finalRef = resolved.Ref
			resolved = next
		}

		// Copy the resolved node
		newCurrent := *resolved
		newCurrent.ResolvedFrom = finalRef
		newCurrent.GoStruct = current.GoStruct
		newCurrent.Embed = current.Embed
		newCurrent.InternalOnly = customInternalOnly

		// Restore custom extensions if they were explicitly set on the reference
		if customGoType != "" {
			newCurrent.GoType = customGoType
		} else if isPrimitiveType(newCurrent.Type) {
			// A primitive definition may use x-customType to choose the underlying
			// Go type for its named declaration. Do not let that underlying type
			// replace the named type at fields that reference the definition.
			newCurrent.GoType = ""
		}
		if customIsPointer {
			newCurrent.IsPointer = customIsPointer
		}
		if customIsOptional {
			newCurrent.IsOptional = customIsOptional
		}
		if customDescription != "" {
			newCurrent.Description = customDescription
		}
		if customDefault != nil {
			newCurrent.Default = customDefault
		}
		if len(customEnum) > 0 {
			newCurrent.Enum = customEnum
		}
		if customMaxLength != nil {
			newCurrent.MaxLength = customMaxLength
		}
		if customMinLength != nil {
			newCurrent.MinLength = customMinLength
		}
		if customPattern != "" {
			newCurrent.Pattern = customPattern
		}
		if customMaximum != nil {
			newCurrent.Maximum = customMaximum
		}
		if customExclusiveMaximum != nil {
			newCurrent.ExclusiveMaximum = customExclusiveMaximum
		}
		if customMinimum != nil {
			newCurrent.Minimum = customMinimum
		}
		if customExclusiveMinimum != nil {
			newCurrent.ExclusiveMinimum = customExclusiveMinimum
		}

		current = &newCurrent
	}

	currRef := reflect.ValueOf(current).Elem()
	targetRef := reflect.ValueOf(target).Elem()

	for i := 0; i < currRef.NumField(); i++ {
		field := currRef.Field(i)
		targetField := targetRef.Field(i)

		if !targetField.CanSet() {
			continue
		}

		switch field.Kind() {
		case reflect.Ptr:
			if !field.IsNil() && field.Elem().Kind() == reflect.Struct {
				if field.Type() == reflect.TypeFor[*ConfigMetadata]() {
					newMeta := &ConfigMetadata{}
					if err := r.resolveSchema(root, field.Interface().(*ConfigMetadata), newMeta, origin); err != nil {
						return err
					}
					targetField.Set(reflect.ValueOf(newMeta))
				}
			} else {
				targetField.Set(field)
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
						if err := r.resolveSchema(root, value.Interface().(*ConfigMetadata), newMeta, origin); err != nil {
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
						if err := r.resolveSchema(root, elem.Interface().(*ConfigMetadata), newMeta, origin); err != nil {
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
	handleEmbeddedStructs(target)
	if err := expandExtendedType(target); err != nil {
		return err
	}
	cleanupInternalDefs(target)
	resolveGoNames(target)

	return nil
}

func isPrimitiveType(typ string) bool {
	switch typ {
	case "string", "integer", "number", "boolean":
		return true
	default:
		return false
	}
}

// resolveRef resolves a JSON Schema $ref, handling both internal and external references.
// The origin parameter tracks which namespace the current schema was loaded from,
// enabling local refs in remotely-fetched schemas to be converted to external refs.
func (r *Resolver) resolveRef(root, current *ConfigMetadata, origin *Ref) (*ConfigMetadata, error) {
	ref := WithOrigin(current.Ref, origin)

	if err := ref.Validate(); err != nil {
		return nil, fmt.Errorf("invalid reference format %q: %w", current.Ref, err)
	}

	if def, ok := lookupRootDef(root, current, ref); ok {
		return def, nil
	}

	if ref.IsLocal() {
		return r.loadExternalRef(ref)
	}

	// check if it's in known namespace
	if _, ok := ref.Namespace(); ok {
		return r.loadExternalRef(ref)
	}

	// fallback to type "any"
	current.GoType = current.Ref
	current.Comment = "Uses `any` type."
	return current, nil
}

func lookupRootDef(root, current *ConfigMetadata, ref *Ref) (*ConfigMetadata, bool) {
	if root.Defs == nil || (!ref.IsInternal() && !ref.IsLocal()) {
		return nil, false
	}
	def, ok := root.Defs[ref.DefName()]
	if !ok || def == current {
		return nil, false
	}
	return def, ok
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

	if md.Defs != nil {
		if def, ok := md.Defs[ref.DefName()]; ok {
			resolved := &ConfigMetadata{}
			if err := r.resolveSchema(md, def, resolved, ref); err != nil {
				return nil, fmt.Errorf("failed to resolve internal references in external schema %s: %w", ref, err)
			}
			return resolved, nil
		}
	}

	return nil, fmt.Errorf("type %q not found in loaded schema for reference %s", ref.DefName(), ref)
}

func resolveGoNames(md *ConfigMetadata) {
	for name, prop := range md.Properties {
		if prop.GoStruct.FieldName == "" {
			prop.GoStruct.FieldName = name
		}
	}
}

func handleEmbeddedStructs(md *ConfigMetadata) {
	embeddedStructs := make([]*ConfigMetadata, 0)
	properties := make(map[string]*ConfigMetadata)
	if len(md.AllOf) > 0 {
		log.Printf("warning: found deprecated allOf, use properties with `embed: true` annotation instead\n")
		embeddedStructs = md.AllOf
	}
	for _, propName := range slices.Sorted(maps.Keys(md.Properties)) {
		prop := md.Properties[propName]
		if prop.Embed {
			if !prop.GoStruct.Anonymous {
				prop.EmbeddedName = propName
			}
			embeddedStructs = append(embeddedStructs, prop)
		} else {
			properties[propName] = prop
		}
	}
	md.AllOf = embeddedStructs
	md.Properties = properties
}

func cleanupInternalDefs(md *ConfigMetadata) {
	for name, def := range md.Defs {
		if def != nil && def.InternalOnly {
			delete(md.Defs, name)
		}
	}
	if len(md.Defs) == 0 {
		md.Defs = nil
	}
}
