// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package schemagen provides JSON schema generation for OpenTelemetry collector component configurations.
// It uses static analysis to extract type information from Go packages without requiring runtime execution.
package schemagen // import "go.opentelemetry.io/collector/cmd/builder/internal/schemagen"

import (
	"encoding/json"
	"fmt"
	"go/types"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"go.opentelemetry.io/collector/component"
)

// SchemaGenerator generates JSON schemas for collector component configurations using static analysis.
type SchemaGenerator struct {
	outputDir string
	analyzer  *PackageAnalyzer
	comments  *CommentExtractor
}

// NewSchemaGenerator creates a new schema generator.
// outputDir is where schema files will be written.
// analyzer is the package analyzer to use for loading types.
func NewSchemaGenerator(outputDir string, analyzer *PackageAnalyzer) *SchemaGenerator {
	return &SchemaGenerator{
		outputDir: outputDir,
		analyzer:  analyzer,
		comments:  NewCommentExtractor(),
	}
}

// ensurePackageComments loads a package and extracts comments if not already done.
// This is needed for types from external packages (embedded types, nested structs).
func (sg *SchemaGenerator) ensurePackageComments(pkgPath string) {
	if pkgPath == "" {
		return
	}
	if pkg, err := sg.analyzer.LoadPackage(pkgPath); err == nil {
		sg.comments.ExtractComments(pkg)
	}
}

// GenerateSchema generates a JSON schema for a component's Config type and writes it to a file.
func (sg *SchemaGenerator) GenerateSchema(kind component.Kind, componentType, importPath string) error {
	pkg, err := sg.analyzer.LoadPackage(importPath)
	if err != nil {
		return fmt.Errorf("failed to load package %s: %w", importPath, err)
	}

	// Extract comments from the package
	sg.comments.ExtractComments(pkg)

	// Find the Config type
	configType, err := sg.analyzer.FindConfigType(pkg)
	if err != nil {
		return fmt.Errorf("failed to find Config type in %s: %w", importPath, err)
	}

	// Generate the schema
	schema, err := sg.generateSchemaFromType(configType, pkg.PkgPath)
	if err != nil {
		return fmt.Errorf("failed to generate schema: %w", err)
	}

	// Write to file
	filename := fmt.Sprintf("%s_%s.json", strings.ToLower(kind.String()), componentType)
	filePath := filepath.Join(sg.outputDir, filename)

	if err := sg.writeSchemaToFile(filePath, schema); err != nil {
		return fmt.Errorf("failed to write schema: %w", err)
	}

	return nil
}

// generateSchemaFromType generates a JSON Schema from a named type.
func (sg *SchemaGenerator) generateSchemaFromType(named *types.Named, pkgPath string) (map[string]any, error) {
	st, ok := GetStructFromNamed(named)
	if !ok {
		return nil, fmt.Errorf("type %s is not a struct", named.Obj().Name())
	}

	schema := map[string]any{
		"$schema":    "https://json-schema.org/draft/2020-12/schema",
		"type":       "object",
		"properties": make(map[string]any),
	}

	properties := schema["properties"].(map[string]any)
	if err := sg.analyzeStructFields(st, properties, named.Obj().Name(), pkgPath); err != nil {
		return nil, err
	}

	return schema, nil
}

// analyzeStructFields recursively analyzes struct fields and populates properties.
func (sg *SchemaGenerator) analyzeStructFields(st *types.Struct, properties map[string]any, typeName, pkgPath string) error {
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		tag := st.Tag(i)

		// Skip unexported fields
		if !field.Exported() {
			continue
		}

		// Handle embedded fields
		if field.Anonymous() {
			if err := sg.handleEmbeddedField(field.Type(), properties); err != nil {
				return fmt.Errorf("failed to handle embedded field %s: %w", field.Name(), err)
			}
			continue
		}

		// Get the field name from tags or use the Go field name
		fieldName := sg.getFieldName(tag, field.Name())
		if fieldName == "" || fieldName == "-" {
			continue
		}

		// Generate property schema
		property, err := sg.generatePropertySchema(field.Type(), tag, field.Name(), typeName, pkgPath)
		if err != nil {
			return fmt.Errorf("failed to generate property schema for %s: %w", field.Name(), err)
		}

		properties[fieldName] = property
	}

	return nil
}

// handleEmbeddedField flattens embedded struct fields into the parent properties.
func (sg *SchemaGenerator) handleEmbeddedField(t types.Type, properties map[string]any) error {
	// Handle pointer to struct
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	// Get the underlying struct
	var st *types.Struct
	var typeName, pkgPath string

	if named, ok := t.(*types.Named); ok {
		if s, ok := named.Underlying().(*types.Struct); ok {
			st = s
			typeName = named.Obj().Name()
			if named.Obj().Pkg() != nil {
				pkgPath = named.Obj().Pkg().Path()
				// Load comments for the embedded type's package
				sg.ensurePackageComments(pkgPath)
			}
		}
	} else if s, ok := t.Underlying().(*types.Struct); ok {
		st = s
	}

	if st == nil {
		return nil
	}

	return sg.analyzeStructFields(st, properties, typeName, pkgPath)
}

// getFieldName extracts the field name from struct tags.
func (sg *SchemaGenerator) getFieldName(tag, goFieldName string) string {
	// Check mapstructure tag first
	if val := extractTagValue(tag, "mapstructure"); val != "" {
		parts := strings.Split(val, ",")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	// Check json tag
	if val := extractTagValue(tag, "json"); val != "" {
		parts := strings.Split(val, ",")
		if len(parts) > 0 && parts[0] != "" && parts[0] != "-" {
			return parts[0]
		}
	}

	// Default to lowercase field name
	return strings.ToLower(goFieldName)
}

// generatePropertySchema generates a JSON Schema property for a field.
func (sg *SchemaGenerator) generatePropertySchema(t types.Type, tag, fieldName, parentTypeName, pkgPath string) (map[string]any, error) {
	property := make(map[string]any)

	// Handle pointer types
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	// Handle configoptional.Optional[T] - unwrap and use inner type
	if innerType, ok := GetOptionalInnerType(t); ok {
		t = innerType
		// Handle pointer inside Optional
		if ptr, ok := t.(*types.Pointer); ok {
			t = ptr.Elem()
		}
	}

	// Check for special types first (Duration, Time, configopaque.String)
	if schema, ok := HandleSpecialType(t); ok {
		property = schema
	} else {
		// Generate schema based on type kind - this properly handles structs
		if err := sg.populateTypeSchema(t, property); err != nil {
			return nil, err
		}
	}

	// Add description from comments
	var description string
	if comment := sg.comments.GetFieldComment(pkgPath, parentTypeName, fieldName); comment != "" {
		description = comment
		property["description"] = comment
	}

	// Check description tag as fallback
	if property["description"] == nil {
		if desc := extractTagValue(tag, "description"); desc != "" {
			description = desc
			property["description"] = desc
		}
	}

	// Check for deprecation
	if IsDeprecatedFromTag(tag) || IsDeprecatedFromDescription(description) {
		property["deprecated"] = true
	}

	return property, nil
}

// populateTypeSchema populates the property map with schema information for a type.
func (sg *SchemaGenerator) populateTypeSchema(t types.Type, property map[string]any) error {
	switch typ := t.(type) {
	case *types.Basic:
		switch typ.Kind() {
		case types.String:
			property["type"] = "string"
		case types.Bool:
			property["type"] = "boolean"
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
			types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
			property["type"] = "integer"
		case types.Float32, types.Float64:
			property["type"] = "number"
		default:
			property["type"] = "object"
		}

	case *types.Slice:
		property["type"] = "array"
		itemSchema := sg.generateTypeSchema(typ.Elem())
		property["items"] = itemSchema

	case *types.Array:
		property["type"] = "array"
		itemSchema := sg.generateTypeSchema(typ.Elem())
		property["items"] = itemSchema

	case *types.Map:
		property["type"] = "object"
		property["additionalProperties"] = true
		if basic, ok := typ.Key().(*types.Basic); ok && basic.Kind() == types.String {
			valueSchema := sg.generateTypeSchema(typ.Elem())
			if len(valueSchema) > 0 {
				property["additionalProperties"] = valueSchema
			}
		}

	case *types.Named:
		// Check if it's an Optional type and unwrap it
		if innerType, ok := GetOptionalInnerType(typ); ok {
			return sg.populateTypeSchema(innerType, property)
		}

		// Check for special types (Duration, Time, configopaque.String)
		if schema, ok := HandleSpecialType(typ); ok {
			maps.Copy(property, schema)
			return nil
		}

		// Handle struct types
		if st, ok := typ.Underlying().(*types.Struct); ok {
			property["type"] = "object"
			nestedProperties := make(map[string]any)
			typeName := typ.Obj().Name()
			pkgPath := ""
			if typ.Obj().Pkg() != nil {
				pkgPath = typ.Obj().Pkg().Path()
				// Load package and extract comments before analyzing fields
				sg.ensurePackageComments(pkgPath)
			}
			if err := sg.analyzeStructFields(st, nestedProperties, typeName, pkgPath); err != nil {
				return err
			}
			if len(nestedProperties) > 0 {
				property["properties"] = nestedProperties
			}
			return nil
		}

		// For other named types, use the underlying type
		return sg.populateTypeSchema(typ.Underlying(), property)

	case *types.Interface:
		property["type"] = "object"
		property["additionalProperties"] = true

	case *types.Struct:
		property["type"] = "object"
		nestedProperties := make(map[string]any)
		if err := sg.analyzeStructFields(typ, nestedProperties, "", ""); err != nil {
			return err
		}
		if len(nestedProperties) > 0 {
			property["properties"] = nestedProperties
		}

	default:
		property["type"] = "object"
	}

	return nil
}

// generateTypeSchema generates a simple schema for a type.
func (sg *SchemaGenerator) generateTypeSchema(t types.Type) map[string]any {
	// Handle pointer types
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	// Handle configoptional.Optional[T] - unwrap and use inner type
	if innerType, ok := GetOptionalInnerType(t); ok {
		return sg.generateTypeSchema(innerType)
	}

	// Check for special types (Duration, Time, configopaque.String)
	if schema, ok := HandleSpecialType(t); ok {
		return schema
	}

	schema := make(map[string]any)

	switch typ := t.(type) {
	case *types.Basic:
		switch typ.Kind() {
		case types.String:
			schema["type"] = "string"
		case types.Bool:
			schema["type"] = "boolean"
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
			types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
			schema["type"] = "integer"
		case types.Float32, types.Float64:
			schema["type"] = "number"
		default:
			schema["type"] = "object"
		}

	case *types.Slice, *types.Array:
		schema["type"] = "array"
		var elemType types.Type
		switch t := typ.(type) {
		case *types.Slice:
			elemType = t.Elem()
		case *types.Array:
			elemType = t.Elem()
		}
		if elemType != nil {
			schema["items"] = sg.generateTypeSchema(elemType)
		}

	case *types.Map:
		schema["type"] = "object"
		schema["additionalProperties"] = true

	case *types.Named:
		// Check if it's an Optional type and unwrap it
		if innerType, ok := GetOptionalInnerType(typ); ok {
			return sg.generateTypeSchema(innerType)
		}

		// Check for special types (Duration, Time, configopaque.String)
		if specialSchema, ok := HandleSpecialType(typ); ok {
			return specialSchema
		}

		st, ok := typ.Underlying().(*types.Struct)
		if !ok {
			return sg.generateTypeSchema(typ.Underlying())
		}
		schema["type"] = "object"
		properties := make(map[string]any)
		typeName := typ.Obj().Name()
		pkgPath := ""
		if typ.Obj().Pkg() != nil {
			pkgPath = typ.Obj().Pkg().Path()
			// Load package and extract comments before analyzing fields
			sg.ensurePackageComments(pkgPath)
		}
		if err := sg.analyzeStructFields(st, properties, typeName, pkgPath); err == nil && len(properties) > 0 {
			schema["properties"] = properties
		}

	case *types.Interface:
		schema["type"] = "object"
		schema["additionalProperties"] = true

	case *types.Struct:
		schema["type"] = "object"
		properties := make(map[string]any)
		if err := sg.analyzeStructFields(typ, properties, "", ""); err == nil && len(properties) > 0 {
			schema["properties"] = properties
		}

	default:
		schema["type"] = "object"
	}

	return schema
}

// writeSchemaToFile writes a schema to a JSON file.
func (sg *SchemaGenerator) writeSchemaToFile(filePath string, schema map[string]any) error {
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schema to JSON: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// OutputDir returns the output directory for schema files.
func (sg *SchemaGenerator) OutputDir() string {
	return sg.outputDir
}

// GetFieldName is a helper function exposed for compatibility with existing tests.
// It extracts the field name from a reflect.StructField.
func GetFieldName(field reflect.StructField) string {
	if tag := field.Tag.Get("mapstructure"); tag != "" {
		parts := strings.Split(tag, ",")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}
	if tag := field.Tag.Get("json"); tag != "" {
		parts := strings.Split(tag, ",")
		if len(parts) > 0 && parts[0] != "" && parts[0] != "-" {
			return parts[0]
		}
	}
	return strings.ToLower(field.Name)
}
