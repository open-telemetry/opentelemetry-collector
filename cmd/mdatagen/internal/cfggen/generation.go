// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/helpers"
)

// NewCfgFns returns template functions for config generation with rootPackage and componentPackage
// baked into closures. This way the template itself never needs to pass these context values around.
func NewCfgFns(rootPackage, componentPackage string) map[string]any {
	return map[string]any{
		"extractImports": func(cfg *ConfigMetadata) []string {
			if cfg == nil {
				return nil
			}
			imports, err := ExtractImports(cfg, rootPackage, componentPackage)
			if err != nil {
				return []string{}
			}
			return imports
		},
		"extractDefs": func(cfg *ConfigMetadata) map[string]*ConfigMetadata {
			if cfg == nil {
				return nil
			}
			return ExtractDefs(cfg)
		},
		"extractValidators": func(cfg *ConfigMetadata) []Validator {
			if cfg == nil {
				return nil
			}
			return ExtractValidators(cfg)
		},
		"mapGoType": func(cfg *ConfigMetadata, propName string) string {
			if cfg == nil {
				return "any"
			}
			goType, err := MapGoType(cfg, propName, rootPackage, componentPackage)
			if err != nil {
				panic(err)
			}
			return goType
		},
		"publicType": func(ref string) string {
			typeName, err := FormatTypeName(ref, rootPackage, componentPackage)
			if err != nil {
				panic(err)
			}
			return typeName
		},
	}
}

// WithCfgFns merges config generation template functions into the given function map.
// The rootPackage and componentPackage are captured in closures so the template doesn't need to thread them through.
func WithCfgFns(fns map[string]any, rootPackage, componentPackage string) map[string]any {
	cfgFns := NewCfgFns(rootPackage, componentPackage)
	maps.Copy(fns, cfgFns)
	return fns
}

var goBasicTypes = []string{
	"rune", "byte",
	"uint", "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64",
	"float32", "float64",
}

// MapGoType maps a ConfigMetadata to its corresponding Go type as a string.
func MapGoType(md *ConfigMetadata, propName, rootPackage, componentPackage string) (string, error) {
	if md == nil {
		return "", errors.New("nil ConfigMetadata")
	}
	goType, err := resolveGoType(md, propName, rootPackage, componentPackage)
	if err != nil {
		return "", fmt.Errorf("failed to resolve Go type for property %q: %w", propName, err)
	}
	if md.IsPointer {
		goType = "*" + goType
	}
	if md.IsOptional {
		return "configoptional.Optional[" + goType + "]", nil
	}
	return goType, nil
}

func resolveGoType(md *ConfigMetadata, propName, rootPackage, componentPackage string) (string, error) {
	if md.GoType != "" {
		if slices.Contains(goBasicTypes, md.GoType) {
			return md.GoType, nil
		}
		typeName, err := FormatTypeName(md.GoType, rootPackage, componentPackage)
		if err != nil {
			return "", fmt.Errorf("failed to format custom type %q: %w", md.GoType, err)
		}
		return typeName, nil
	}
	if md.Ref != "" {
		typeName, err := FormatTypeName(md.Ref, rootPackage, componentPackage)
		if err != nil {
			return "", fmt.Errorf("failed to format reference type %q: %w", md.Ref, err)
		}
		return typeName, nil
	}

	switch md.Type {
	case "string":
		switch md.Format {
		case "date-time":
			return "time.Time", nil
		case "duration":
			return "time.Duration", nil
		default:
			return "string", nil
		}
	case "integer":
		return "int", nil
	case "number":
		return "float64", nil
	case "boolean":
		return "bool", nil
	case "array":
		if md.Items == nil {
			return "[]any", nil
		}
		itemType, err := MapGoType(md.Items, propName+"_item", rootPackage, componentPackage)
		if err != nil {
			return "", fmt.Errorf("failed to map array item type: %w", err)
		}
		return "[]" + itemType, nil
	case "object":
		if md.AdditionalProperties != nil {
			valueType, err := MapGoType(md.AdditionalProperties, propName, rootPackage, componentPackage)
			if err != nil {
				return "", fmt.Errorf("failed to map additionalProperties type: %w", err)
			}
			return "map[string]" + valueType, nil
		}
		if md.Properties != nil {
			formatted, err := helpers.FormatIdentifier(propName, true)
			if err != nil {
				return "", fmt.Errorf("failed to format embedded object type name %q: %w", propName, err)
			}
			return formatted, nil
		}
		return "map[string]any", nil
	case "":
		return "any", nil
	default:
		return "", fmt.Errorf("unsupported type: %q", md.Type)
	}
}

// ExtractImports recursively scans the ConfigMetadata and collects all unique import paths needed for the generated Go code.
func ExtractImports(md *ConfigMetadata, rootPackage, componentPackage string) ([]string, error) {
	if md == nil {
		return nil, nil
	}

	imports := make(map[string]bool)
	if err := collectImports(md, imports, rootPackage, componentPackage); err != nil {
		return nil, err
	}

	return slices.Collect(maps.Keys(imports)), nil
}

func collectImports(md *ConfigMetadata, imports map[string]bool, rootPackage, componentPackage string) error {
	if md == nil {
		return nil
	}

	if md.GoType != "" {
		ref, err := ResolveGoTypeRef(md.GoType, rootPackage, componentPackage)
		if err == nil && ref.ImportPath != "" {
			imports[ref.ImportPath] = true
		}
	}

	if md.Ref != "" {
		ref, err := ResolveGoTypeRef(md.Ref, rootPackage, componentPackage)
		if err == nil && ref.ImportPath != "" {
			imports[ref.ImportPath] = true
		}
	}

	if md.Type == "string" && (md.Format == "date-time" || md.Format == "duration") {
		imports["time"] = true
	}

	if md.IsOptional {
		imports["go.opentelemetry.io/collector/config/configoptional"] = true
	}

	for _, prop := range md.Properties {
		if err := collectImports(prop, imports, rootPackage, componentPackage); err != nil {
			return err
		}
	}

	if md.Items != nil {
		if err := collectImports(md.Items, imports, rootPackage, componentPackage); err != nil {
			return err
		}
	}

	for _, schema := range md.AllOf {
		if err := collectImports(schema, imports, rootPackage, componentPackage); err != nil {
			return err
		}
	}

	for _, def := range md.Defs {
		if err := collectImports(def, imports, rootPackage, componentPackage); err != nil {
			return err
		}
	}

	if err := collectImports(md.AdditionalProperties, imports, rootPackage, componentPackage); err != nil {
		return err
	}

	if md.ContentSchema != nil {
		if err := collectImports(md.ContentSchema, imports, rootPackage, componentPackage); err != nil {
			return err
		}
	}

	return nil
}

// FormatTypeName resolves a reference string to a Go type expression using GoTypeRef.
func FormatTypeName(ref, rootPackage, componentPackage string) (string, error) {
	tr, err := ResolveGoTypeRef(ref, rootPackage, componentPackage)
	if err != nil {
		return "", err
	}
	return tr.String(), nil
}

// ExtractDefs recursively collects all definitions from the ConfigMetadata, including nested ones,
// and returns a flat map of definition names to their corresponding ConfigMetadata.
func ExtractDefs(md *ConfigMetadata) map[string]*ConfigMetadata {
	defs := make(map[string]*ConfigMetadata)
	collectDefs(md, defs)
	return defs
}

func collectDefs(md *ConfigMetadata, defs map[string]*ConfigMetadata) {
	if md == nil {
		return
	}

	for _, name := range slices.Sorted(maps.Keys(md.Defs)) {
		defs[name] = md.Defs[name]
		collectDefs(md.Defs[name], defs)
	}

	for _, propName := range slices.Sorted(maps.Keys(md.Properties)) {
		collectDefsForSchema(propName, md.Properties[propName], defs)
	}

	for _, schema := range md.AllOf {
		collectDefs(schema, defs)
	}
}

func collectDefsForSchema(propName string, md *ConfigMetadata, defs map[string]*ConfigMetadata) {
	if md == nil || md.Ref != "" || md.GoType != "" {
		return
	}

	switch md.Type {
	case "object":
		if len(md.Properties) > 0 {
			defs[propName] = md
			collectDefs(md, defs)
		} else if md.AdditionalProperties != nil {
			// map[string]V — the value type V inherits the same propName
			collectDefsForSchema(propName, md.AdditionalProperties, defs)
		}
	case "array":
		if md.Items != nil {
			// []T — item type uses propName+"_item", matching MapGoType
			collectDefsForSchema(propName+"_item", md.Items, defs)
		}
	}
}

// ExtractValidators recursively scans the ConfigMetadata and collects validators for required fields and nested schemas.
func ExtractValidators(md *ConfigMetadata) []Validator {
	validators := make([]Validator, 0)

	if md == nil {
		return validators
	}
	collectValidators(md, &validators)
	slices.SortFunc(validators, func(a, b Validator) int {
		return cmp.Compare(a.FieldName, b.FieldName)
	})

	return validators
}

type Validator struct {
	FieldName       string
	FieldType       string
	IsRequired      bool
	IsPointer       bool
	IsOptional      bool
	CustomValidator string
}

func collectValidators(md *ConfigMetadata, validators *[]Validator) {
	for _, propName := range slices.Sorted(maps.Keys(md.Properties)) {
		prop := md.Properties[propName]
		isRequired := slices.Contains(md.Required, propName)
		if isRequired {
			*validators = append(*validators, Validator{
				FieldName:  propName,
				FieldType:  resolveType(prop),
				IsRequired: isRequired,
				IsPointer:  prop.IsPointer,
				IsOptional: prop.IsOptional,
			})
		}
		if prop.GoStruct.CustomValidator != nil {
			*validators = append(*validators, Validator{
				FieldName:       propName,
				FieldType:       resolveType(prop),
				IsPointer:       prop.IsPointer,
				IsOptional:      prop.IsOptional,
				CustomValidator: generateValidatorName(propName, prop.GoStruct.CustomValidator),
			})
		}
	}

	if md.GoStruct.CustomValidator != nil {
		*validators = append(*validators, Validator{
			FieldName:       ".",
			FieldType:       md.Type,
			CustomValidator: generateValidatorName("", md.GoStruct.CustomValidator),
		})
	}
}

func resolveType(md *ConfigMetadata) string {
	switch {
	case md.Ref != "":
		return "ref"
	case md.Type == "string" && md.Format == "date-time":
		return "datetime"
	case md.Type == "string" && md.Format == "duration":
		return "duration"
	case md.Type == "object" && md.AdditionalProperties != nil:
		return "map"
	default:
		return md.Type
	}
}

func generateValidatorName(propName string, desc *CustomValidatorConfig) string {
	if desc.Name != "" {
		return desc.Name
	}
	id, _ := helpers.FormatIdentifier(propName, true)
	return "validate" + id
}
