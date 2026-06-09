// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/internal/schemagen"

import (
	"errors"
	"fmt"
	"log"
	"maps"
	"reflect"
	"slices"
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

// Resolve takes the source Metadata parsed from metadata.yaml and produces a
// JSONSchemaDoc with all $ref references resolved. exported_configs from the source
// become the top-level $defs of the output document.
//
// The source Metadata is treated as read-only: the inputs are deep-cloned before the
// recursive resolver walks them, so internal mutations (the "any" fallback in
// resolveRef, per-node type writes) cannot leak back into the caller's tree through
// shared *ConfigMetadata pointers.
func (r *Resolver) Resolve(src *Metadata) (*JSONSchemaDoc, error) {
	if src == nil {
		return nil, errors.New("nil metadata")
	}
	work := src.Clone()
	config := work.Config
	if config == nil {
		config = &ConfigMetadata{}
	}

	// For ref-lookup purposes the recursive resolver expects the top-level
	// definitions (both nested $defs and exported_configs) to be reachable from
	// the root node. Merge them onto the owned root. On collision the nested
	// $defs win: the legacy mdatagen path injected internal generated defs over
	// exported_configs, so a generated name like metrics_builder_config must
	// resolve to the internal def, not an exported config sharing the name.
	if len(work.ExportedConfigs) > 0 {
		merged := make(map[string]*ConfigMetadata, len(work.ExportedConfigs)+len(config.Defs))
		maps.Copy(merged, work.ExportedConfigs)
		maps.Copy(merged, config.Defs)
		config.Defs = merged
	}

	target := &ConfigMetadata{}
	if err := r.resolveSchema(config, config, target, nil); err != nil {
		return nil, err
	}

	target.Schema = schemaVersion
	target.ID = r.pkgID
	target.Title = fmt.Sprintf("%s/%s", r.class, r.name)
	if len(config.Properties) > 0 {
		target.Type = "object"
	}

	doc := &JSONSchemaDoc{ConfigMetadata: target}
	// Defs on the resolved target are the merged exported_configs and any internal
	// $defs that survived the resolver's cleanup pass. Promote them to the document's
	// $defs and clear them off the inner node so they are not serialized twice.
	if len(target.Defs) > 0 {
		doc.Defs = target.Defs
		target.Defs = nil
	}
	return doc, nil
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
	enhanceTimeTypes(target)
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
	m, err := r.loader.Load(*ref)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, fmt.Errorf("no loader could resolve external reference: %s", ref)
	}

	// Loaders may cache and reuse the same *Metadata across calls, so deep-clone
	// here. Otherwise mutations performed by the recursive resolveSchema (the "any"
	// fallback in resolveRef, per-node type writes) would persist into the cache
	// and contaminate subsequent loads.
	m = m.Clone()
	root := m.Config
	if root == nil {
		root = &ConfigMetadata{}
	}
	if len(m.ExportedConfigs) > 0 {
		merged := make(map[string]*ConfigMetadata, len(m.ExportedConfigs)+len(root.Defs))
		maps.Copy(merged, m.ExportedConfigs)
		maps.Copy(merged, root.Defs)
		root.Defs = merged
	}

	if def, ok := root.Defs[ref.DefName()]; ok {
		resolved := &ConfigMetadata{}
		if err := r.resolveSchema(root, def, resolved, ref); err != nil {
			return nil, fmt.Errorf("failed to resolve internal references in external schema %s: %w", ref, err)
		}
		return resolved, nil
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

func enhanceTimeTypes(md *ConfigMetadata) {
	if md.Type == "string" {
		switch md.Format {
		case "duration":
			md.Format = ""
			md.GoType = "time.Duration"
			md.Pattern = goDurationPattern
		case "date-time":
			md.GoType = "time.Time"
		}
	}
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
