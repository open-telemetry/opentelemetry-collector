// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// This file is a self-contained spike for #15315 path B: can the writer-side JSON
// Schema representation be a struct provided by a third-party library
// (github.com/invopop/jsonschema) instead of a project-owned type? It does not
// replace the existing writer; it only demonstrates the mapping and surfaces the
// fit issues that show up in practice. Companion to PR opentelemetry-collector#15346
// (path A, internal JSONSchemaDoc).

package schemagen // import "go.opentelemetry.io/collector/internal/schemagen"

import (
	"encoding/json"
	"strconv"

	"github.com/invopop/jsonschema"
)

// ToInvopopSchema converts a source Metadata into a github.com/invopop/jsonschema
// Schema. The mapping covers the JSON Schema 2020-12 fields the project actually
// emits today. Fields that have no native equivalent on the third-party type are
// surfaced via Schema.Extras, which the library marshals at the schema root.
//
// Caveats this spike intentionally exposes:
//
//   - Properties on invopop's Schema is an *orderedmap.OrderedMap, not a Go map.
//     That changes how callers construct and iterate properties (the resolver and
//     the mdatagen templates would need to migrate).
//   - The project-specific x-* extensions (x-customType, x-pointer, x-optional)
//     and the go_struct / embed metadata fields have no slot on the library type.
//     They go through Extras and lose their typed Go representation; reads after
//     unmarshal would need to assert types out of map[string]any.
//   - InternalOnly, ResolvedFrom, EmbeddedName are not JSON Schema concepts; they
//     are resolver state and are deliberately dropped here.
//   - Numeric fields (multipleOf/maximum/minimum) on invopop are json.Number; our
//     model uses *float64. The conversion has to format-and-parse rather than copy.
func ToInvopopSchema(m *Metadata) *jsonschema.Schema {
	if m == nil {
		return nil
	}
	out := schemaNodeToInvopop(m.Config)
	if out == nil {
		out = &jsonschema.Schema{}
	}
	if len(m.ExportedConfigs) > 0 {
		out.Definitions = make(jsonschema.Definitions, len(m.ExportedConfigs))
		for k, v := range m.ExportedConfigs {
			out.Definitions[k] = schemaNodeToInvopop(v)
		}
	}
	return out
}

// schemaNodeToInvopop maps one ConfigMetadata node to one invopop *Schema. The
// recursion sites mirror the original struct's pointer/map/slice shape.
func schemaNodeToInvopop(c *ConfigMetadata) *jsonschema.Schema {
	if c == nil {
		return nil
	}
	s := &jsonschema.Schema{
		Version:          c.Schema,
		ID:               jsonschema.ID(c.ID),
		Title:            c.Title,
		Description:      c.Description,
		Comments:         c.Comment,
		Type:             c.Type,
		Ref:              c.Ref,
		Default:          c.Default,
		Examples:         c.Examples,
		Deprecated:       c.Deprecated,
		Enum:             c.Enum,
		Const:            c.Const,
		Required:         c.Required,
		Pattern:          c.Pattern,
		Format:           c.Format,
		ContentMediaType: c.ContentMediaType,
		ContentEncoding:  c.ContentEncoding,
		UniqueItems:      c.UniqueItems,
	}

	// Numeric fields: project model is *float64, library is json.Number. The
	// stringly-typed nature forces a format/parse round trip rather than a copy.
	if c.MultipleOf != nil {
		s.MultipleOf = json.Number(strconv.FormatFloat(*c.MultipleOf, 'f', -1, 64))
	}
	if c.Maximum != nil {
		s.Maximum = json.Number(strconv.FormatFloat(*c.Maximum, 'f', -1, 64))
	}
	if c.ExclusiveMaximum != nil {
		s.ExclusiveMaximum = json.Number(strconv.FormatFloat(*c.ExclusiveMaximum, 'f', -1, 64))
	}
	if c.Minimum != nil {
		s.Minimum = json.Number(strconv.FormatFloat(*c.Minimum, 'f', -1, 64))
	}
	if c.ExclusiveMinimum != nil {
		s.ExclusiveMinimum = json.Number(strconv.FormatFloat(*c.ExclusiveMinimum, 'f', -1, 64))
	}

	// Length / item bounds: library is *uint64, project is *int.
	if c.MinLength != nil {
		v := uint64(*c.MinLength)
		s.MinLength = &v
	}
	if c.MaxLength != nil {
		v := uint64(*c.MaxLength)
		s.MaxLength = &v
	}
	if c.MinItems != nil {
		v := uint64(*c.MinItems)
		s.MinItems = &v
	}
	if c.MaxItems != nil {
		v := uint64(*c.MaxItems)
		s.MaxItems = &v
	}
	if c.MinProperties != nil {
		v := uint64(*c.MinProperties)
		s.MinProperties = &v
	}
	if c.MaxProperties != nil {
		v := uint64(*c.MaxProperties)
		s.MaxProperties = &v
	}

	if c.Items != nil {
		s.Items = schemaNodeToInvopop(c.Items)
	}
	if c.AdditionalProperties != nil {
		s.AdditionalProperties = schemaNodeToInvopop(c.AdditionalProperties)
	}
	if c.ContentSchema != nil {
		s.ContentSchema = schemaNodeToInvopop(c.ContentSchema)
	}
	if len(c.AllOf) > 0 {
		s.AllOf = make([]*jsonschema.Schema, 0, len(c.AllOf))
		for _, sub := range c.AllOf {
			s.AllOf = append(s.AllOf, schemaNodeToInvopop(sub))
		}
	}
	if len(c.Properties) > 0 {
		s.Properties = jsonschema.NewProperties()
		for k, v := range c.Properties {
			s.Properties.Set(k, schemaNodeToInvopop(v))
		}
	}
	if len(c.Defs) > 0 && s.Definitions == nil {
		s.Definitions = make(jsonschema.Definitions, len(c.Defs))
		for k, v := range c.Defs {
			s.Definitions[k] = schemaNodeToInvopop(v)
		}
	}

	// The project-specific extensions have no slot on jsonschema.Schema. Carry them
	// through Extras, which the library marshals at the schema root. Note that the
	// typed Go shape (e.g. GoStructConfig struct, IsPointer bool) collapses into
	// untyped any here; readers would have to assert types back out.
	extras := map[string]any{}
	if c.GoType != "" {
		extras["x-customType"] = c.GoType
	}
	if c.IsPointer {
		extras["x-pointer"] = true
	}
	if c.IsOptional {
		extras["x-optional"] = true
	}
	if c.Embed {
		extras["embed"] = true
	}
	if c.GoStruct != (GoStructConfig{}) {
		extras["go_struct"] = c.GoStruct
	}
	if len(extras) > 0 {
		s.Extras = extras
	}

	return s
}
