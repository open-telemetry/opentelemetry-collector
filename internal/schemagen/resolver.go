// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/internal/schemagen"

import (
	"fmt"
)

const (
	// goDurationPattern matches Go duration strings (e.g., "30s", "1h30m", "500ms")
	goDurationPattern = `^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`
)

type Resolver struct {
	loader Loader
}

func NewResolver(dir string) *Resolver {
	loader := NewLoader(dir)

	return &Resolver{
		loader: loader,
	}
}

// ResolveSchema takes a source configuration metadata schema and resolves all references ($ref)
// to produce a fully resolved schema. It handles both internal references (within the same schema) and external references
// (pointing to other schemas, either locally or remotely). The resolver uses registered loaders to fetch external schemas as needed.
//
// Returns a new ConfigMetadata with all references resolved, or an error if resolution fails.
func (r *Resolver) ResolveSchema(src *ConfigsMetadata) (*ConfigsMetadata, error) {
	target := src.Clone()
	if src.ExportedConfigs != nil {
		for _, config := range target.ExportedConfigs {
			err := r.resolveSchema(src, config, nil)
			if err != nil {
				return nil, err
			}
		}
	}
	if src.Config != nil {
		err := r.resolveSchema(src, target.Config, nil)
		if err != nil {
			return nil, err
		}
	}

	return target, nil
}

func (r *Resolver) resolveSchema(root *ConfigsMetadata, target *ConfigMetadata, origin *Ref) error {
	if len(target.Properties) > 0 {
		target.Type = "object"
	}

	if target.Ref != "" {
		ref := target.Ref
		resolved, err := r.resolveRef(root, target, origin)
		if err != nil {
			return fmt.Errorf("failed to resolve $ref %q: %w", ref, err)
		}

		// merge resolved node
		target.MergeFrom(resolved)

		return nil
	}

	for propName, prop := range target.Properties {
		if prop.GoStruct.FieldName == "" {
			prop.GoStruct.FieldName = propName
		}
		if err := r.resolveSchema(root, prop, origin); err != nil {
			return err
		}
	}

	if target.AdditionalProperties != nil {
		if err := r.resolveSchema(root, target.AdditionalProperties, origin); err != nil {
			return err
		}
	}

	if target.Items != nil {
		if err := r.resolveSchema(root, target.Items, origin); err != nil {
			return err
		}
	}

	enhanceTimeTypes(target)

	return nil
}

// resolveRef resolves a JSON Schema $ref, handling both internal and external references.
// The origin parameter tracks which namespace the current schema was loaded from,
// enabling local refs in remotely-fetched schemas to be converted to external refs.
func (r *Resolver) resolveRef(root *ConfigsMetadata, md *ConfigMetadata, origin *Ref) (*ConfigMetadata, error) {
	ref := WithOrigin(md.Ref, origin)

	if err := ref.Validate(); err != nil {
		return nil, fmt.Errorf("invalid reference format %q: %w", md.Ref, err)
	}
	if def, ok := lookupRootDef(root, ref); ok {
		if err := r.resolveSchema(root, def, ref); err != nil {
			return nil, fmt.Errorf("failed to resolve internal references in local schema %s: %w", ref, err)
		}
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
	md.GoType = md.Ref
	return md, nil
}

func lookupRootDef(root *ConfigsMetadata, ref *Ref) (*ConfigMetadata, bool) {
	if root.ExportedConfigs == nil || ref.IsExternal() {
		return nil, false
	}
	def, ok := root.ExportedConfigs[ref.ConfigName()]
	if !ok {
		return nil, false
	}
	if ref.IsLocal() && !def.InternalOnly {
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

	if md.ExportedConfigs != nil {
		if def, ok := md.ExportedConfigs[ref.ConfigName()]; ok {
			resolved := def.Clone()
			if err := r.resolveSchema(md, resolved, ref); err != nil {
				return nil, fmt.Errorf("failed to resolve internal references in external schema %s: %w", ref, err)
			}
			return resolved, nil
		}
	}

	return nil, fmt.Errorf("type %q not found in loaded schema for reference %s", ref.ConfigName(), ref)
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
