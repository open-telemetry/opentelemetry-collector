// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"
import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

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
		"mapDefaultValue": func(cfg *ConfigMetadata) string {
			if cfg == nil || cfg.Default == nil {
				return ""
			}
			val, err := MapDefaultValue(cfg)
			if err != nil {
				panic(err)
			}
			return val
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

// MapDefaultValue Maps a ConfigMetadata default value to its corresponding Go textual representation.
func MapDefaultValue(md *ConfigMetadata) (string, error) {
	if md == nil || md.Default == nil {
		return "", errors.New("nil ConfigMetadata or Default")
	}

	if md.Ref != "" || md.GoType != "" {
		// Custom types or refs might not have an easy textual representation of their literal
		// unless they are primitive go types or durations, but let's try our best.
		// For strings/durations:
		if md.Type == "string" && md.Format == "duration" {
			d, err := time.ParseDuration(fmt.Sprintf("%v", md.Default))
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("time.Duration(%d) /* %s */", d, md.Default), nil
		}
	}

	switch md.Type {
	case "string":
		if md.Format == "duration" {
			d, err := time.ParseDuration(fmt.Sprintf("%v", md.Default))
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("time.Duration(%d) /* %s */", d, md.Default), nil
		}
		return fmt.Sprintf("%q", md.Default), nil
	case "integer":
		return fmt.Sprintf("%v", md.Default), nil
	case "number":
		return fmt.Sprintf("%v", md.Default), nil
	case "boolean":
		return fmt.Sprintf("%v", md.Default), nil
	case "array":
		// Handle default for slices? Usually they're represented as Go slices.
		// For now we can print them if they're simple literals
		return fmt.Sprintf("%#v", md.Default), nil
	default:
		return "", fmt.Errorf("unsupported type for default value: %q", md.Type)
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

	for name, def := range md.Defs {
		defs[name] = def
		collectDefs(def, defs)
	}

	for propName, prop := range md.Properties {
		// if is embedded object
		if prop.Type == "object" {
			if len(prop.Properties) > 0 {
				defs[propName] = prop
				collectDefs(prop, defs)
			}
			ap := md.AdditionalProperties
			if ap != nil && ap.Type == "object" && len(ap.Properties) > 0 {
				defs[propName] = ap
				collectDefs(ap, defs)
			}
		}
		if prop.Type == "array" {
			if prop.Items != nil && prop.Items.Type == "object" && len(prop.Items.Properties) > 0 {
				defName := propName + "_item"
				defs[defName] = prop.Items
				collectDefs(prop.Items, defs)
			}
		}
	}
}
