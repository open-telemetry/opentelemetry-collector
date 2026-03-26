// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"
import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
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
		"extractDefaults": func(cfg *ConfigMetadata) []DefaultValue {
			if cfg == nil {
				return nil
			}
			return ExtractDefaults(cfg)
		},
		"extractDefRefs": func(cfg *ConfigMetadata) []DefRef {
			if cfg == nil {
				return nil
			}
			return ExtractDefRefs(cfg)
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

	for name, def := range md.Defs {
		defs[name] = def
		collectDefs(def, defs)
	}

	for propName, prop := range md.Properties {
		collectDefsForSchema(propName, prop, defs)
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

// DefaultValue represents a resolved default expression for a single Config field.
type DefaultValue struct {
	FieldName  string // original snake_case property name
	GoExpr     string // Go literal expression, e.g. `"localhost:4317"` or `10*time.Second`
	IsOptional bool   // true when x-optional: true → wraps with configoptional.Some(...)
	IsPointer  bool   // true when x-pointer: true → emits a local var and takes its address
}

// ExtractDefaults scans the top-level properties of md and collects fields that have a
// `default:` value expressible as a Go literal. Results are sorted by field name for
// deterministic output. Skips: x-customType, $ref, object, array, date-time fields.
func ExtractDefaults(md *ConfigMetadata) []DefaultValue {
	if md == nil {
		return nil
	}
	var defaults []DefaultValue
	for _, propName := range slices.Sorted(maps.Keys(md.Properties)) {
		prop := md.Properties[propName]
		if prop == nil || prop.Default == nil {
			continue
		}
		if dv, ok := renderDefault(propName, prop); ok {
			defaults = append(defaults, dv)
		}
	}
	return defaults
}

func renderDefault(propName string, md *ConfigMetadata) (DefaultValue, bool) {
	if md.GoType != "" || md.Ref != "" {
		return DefaultValue{}, false
	}
	if md.Type == "object" || md.Type == "array" {
		return DefaultValue{}, false
	}
	if md.Type == "string" && md.Format == "date-time" {
		return DefaultValue{}, false
	}

	var goExpr string
	switch md.Type {
	case "string":
		if md.Format == "duration" {
			expr, ok := renderDurationExpr(md.Default)
			if !ok {
				return DefaultValue{}, false
			}
			goExpr = expr
		} else {
			s, ok := md.Default.(string)
			if !ok {
				return DefaultValue{}, false
			}
			goExpr = fmt.Sprintf("%q", s)
		}
	case "integer":
		switch v := md.Default.(type) {
		case int:
			goExpr = strconv.Itoa(v)
		case int64:
			goExpr = strconv.FormatInt(v, 10)
		case float64:
			goExpr = strconv.Itoa(int(v))
		default:
			return DefaultValue{}, false
		}
	case "number":
		switch v := md.Default.(type) {
		case float64:
			goExpr = strconv.FormatFloat(v, 'f', -1, 64)
		case int:
			goExpr = strconv.FormatFloat(float64(v), 'f', -1, 64)
		default:
			return DefaultValue{}, false
		}
	case "boolean":
		switch v := md.Default.(type) {
		case bool:
			if v {
				goExpr = "true"
			} else {
				goExpr = "false"
			}
		default:
			return DefaultValue{}, false
		}
	default:
		return DefaultValue{}, false
	}

	return DefaultValue{
		FieldName:  propName,
		GoExpr:     goExpr,
		IsOptional: md.IsOptional,
		IsPointer:  md.IsPointer,
	}, true
}

func renderDurationExpr(value any) (string, bool) {
	s, ok := value.(string)
	if !ok {
		return "", false
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return "", false
	}
	return formatDurationAsGoExpr(d), true
}

func formatDurationAsGoExpr(d time.Duration) string {
	if d == 0 {
		return "0"
	}
	units := []struct {
		name  string
		value time.Duration
	}{
		{"time.Hour", time.Hour},
		{"time.Minute", time.Minute},
		{"time.Second", time.Second},
		{"time.Millisecond", time.Millisecond},
		{"time.Microsecond", time.Microsecond},
		{"time.Nanosecond", time.Nanosecond},
	}
	var parts []string
	rem := d
	for _, u := range units {
		if rem >= u.value {
			n := rem / u.value
			rem -= n * u.value
			parts = append(parts, fmt.Sprintf("%d*%s", n, u.name))
		}
	}
	return strings.Join(parts, " + ")
}

type DefRef struct {
	FieldName  string
	DefName    string
	IsPointer  bool
	IsArray    bool
	IsRequired bool
}

// ExtractDefRefs recursively scans the ConfigMetadata for properties that have `default` specified or
// reference nested definitions (defs) with defaults. It also includes allOf entries that are plain
// internal def references whose def has defaults.
func ExtractDefRefs(md *ConfigMetadata) []DefRef {
	if md == nil {
		return nil
	}
	defs := ExtractDefs(md)
	var refs []DefRef
	for _, propName := range slices.Sorted(maps.Keys(md.Properties)) {
		prop := md.Properties[propName]
		if prop == nil {
			continue
		}
		defName, isArray := resolveDefName(propName, prop)
		if defName == "" {
			continue
		}
		defMeta, ok := defs[defName]
		if !ok {
			continue
		}
		if !defHasDefaults(defMeta) {
			continue
		}
		refs = append(refs, DefRef{
			FieldName:  propName,
			DefName:    defName,
			IsPointer:  prop.IsPointer,
			IsArray:    isArray,
			IsRequired: slices.Contains(md.Required, propName),
		})
	}
	// allOf entries: plain internal $ref → FieldName = DefName = ref value
	for _, schema := range md.AllOf {
		if schema.Ref == "" || strings.ContainsAny(schema.Ref, "./") {
			continue
		}
		defMeta, ok := defs[schema.Ref]
		if !ok || !defHasDefaults(defMeta) {
			continue
		}
		refs = append(refs, DefRef{
			FieldName:  schema.Ref,
			DefName:    schema.Ref,
			IsPointer:  false,
			IsArray:    false,
			IsRequired: false,
		})
	}
	return refs
}

func defHasDefaults(md *ConfigMetadata) bool {
	if len(ExtractDefaults(md)) > 0 {
		return true
	}
	defs := ExtractDefs(md)
	for propName, prop := range md.Properties {
		defName, _ := resolveDefName(propName, prop)
		if defName == "" {
			continue
		}
		nested, ok := defs[defName]
		if ok && defHasDefaults(nested) {
			return true
		}
	}
	return false
}

func resolveDefName(propName string, md *ConfigMetadata) (defName string, isArray bool) {
	if md == nil || md.GoType != "" {
		return "", false
	}
	// Plain internal $ref (no "/" or ".") — the ref value is the def name directly.
	if md.Ref != "" {
		if !strings.ContainsAny(md.Ref, "./") {
			return md.Ref, false
		}
		return "", false
	}
	switch md.Type {
	case "object":
		if len(md.Properties) > 0 {
			return propName, false
		}
	case "array":
		if md.Items != nil && md.Items.Ref == "" && md.Items.GoType == "" &&
			md.Items.Type == "object" && len(md.Items.Properties) > 0 {
			return propName + "_item", true
		}
	}
	return "", false
}

// ExtractValidators recursively scans the ConfigMetadata and collects validators for required fields and nested schemas.
func ExtractValidators(md *ConfigMetadata) []Validator {
	validators := make([]Validator, 0)

	if md == nil {
		return validators
	}
	collectValidators(md, &validators)

	return validators
}

type Validator struct {
	FieldName  string
	FieldType  string
	IsRequired bool
	IsPointer  bool
	IsOptional bool
}

func collectValidators(md *ConfigMetadata, validators *[]Validator) {
	for propName, prop := range md.Properties {
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
	default:
		return md.Type
	}
}
