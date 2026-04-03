// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
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
		"extractDefaults": func(cfg *ConfigMetadata) []*DefaultAssigment {
			if cfg == nil {
				return nil
			}
			return ExtractDefaults(cfg)
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
	"time.Time", "time.Duration",
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
	if md.ResolvedFrom != "" {
		typeName, err := FormatTypeName(md.ResolvedFrom, rootPackage, componentPackage)
		if err != nil {
			return "", fmt.Errorf("failed to format reference type %q: %w", md.ResolvedFrom, err)
		}
		return typeName, nil
	}

	switch md.Type {
	case "string":
		if strings.HasPrefix(md.GoType, "time.") {
			return md.GoType, nil
		}
		return "string", nil
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

	if md.Type == "string" && strings.HasPrefix(md.GoType, "time.") {
		imports["time"] = true
	}

	if md.IsOptional {
		imports["go.opentelemetry.io/collector/config/configoptional"] = true
	}

	if md.ResolvedFrom != "" {
		ref, err := ResolveGoTypeRef(md.ResolvedFrom, rootPackage, componentPackage)
		if err == nil && ref.ImportPath != "" {
			imports[ref.ImportPath] = true
		}
		collectDefaultImports(md, md.Default, imports, rootPackage, componentPackage)
		return nil
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

func collectDefaultImports(md *ConfigMetadata, defaultValue any, imports map[string]bool, rootPackage, componentPackage string) {
	if md == nil || !hasDefaultInTree(md, defaultValue) {
		return
	}

	if md.GoType != "" {
		ref, err := ResolveGoTypeRef(md.GoType, rootPackage, componentPackage)
		if err == nil && ref.ImportPath != "" {
			imports[ref.ImportPath] = true
		}
	}

	if md.ResolvedFrom != "" {
		ref, err := ResolveGoTypeRef(md.ResolvedFrom, rootPackage, componentPackage)
		if err == nil && ref.ImportPath != "" {
			imports[ref.ImportPath] = true
		}
	}

	if md.Type == "string" && strings.HasPrefix(md.GoType, "time.") {
		imports["time"] = true
	}

	if md.IsOptional {
		imports["go.opentelemetry.io/collector/config/configoptional"] = true
	}

	overrides, _ := defaultValue.(map[string]any)

	for _, propName := range slices.Sorted(maps.Keys(md.Properties)) {
		prop := md.Properties[propName]
		effectiveDefault := prop.Default
		if overrides != nil {
			if v, ok := overrides[propName]; ok {
				effectiveDefault = v
			}
		}
		collectDefaultImports(prop, effectiveDefault, imports, rootPackage, componentPackage)
	}

	for _, schema := range md.AllOf {
		collectDefaultImports(schema, schema.Default, imports, rootPackage, componentPackage)
	}

	if arr, ok := defaultValue.([]any); ok && md.Items != nil {
		for _, item := range arr {
			collectDefaultImports(md.Items, item, imports, rootPackage, componentPackage)
		}
	}

	if obj, ok := defaultValue.(map[string]any); ok && md.AdditionalProperties != nil && isMapOfObjects(md) {
		for _, value := range obj {
			collectDefaultImports(md.AdditionalProperties, value, imports, rootPackage, componentPackage)
		}
	}
}

func hasDefaultInTree(md *ConfigMetadata, defaultValue any) bool {
	if md == nil {
		return false
	}

	switch {
	case isObjectWithProperties(md):
		overrides, _ := defaultValue.(map[string]any)
		for _, propName := range slices.Sorted(maps.Keys(md.Properties)) {
			prop := md.Properties[propName]
			effectiveDefault := prop.Default
			if overrides != nil {
				if v, ok := overrides[propName]; ok {
					effectiveDefault = v
				}
			}
			if hasDefaultInTree(prop, effectiveDefault) {
				return true
			}
		}
		for _, schema := range md.AllOf {
			if hasDefaultInTree(schema, schema.Default) {
				return true
			}
		}
		return false
	case isArrayOfObjects(md):
		arr, _ := defaultValue.([]any)
		return len(arr) > 0
	case isMapOfObjects(md):
		obj, _ := defaultValue.(map[string]any)
		return len(obj) > 0
	default:
		return defaultValue != nil
	}
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
	if md == nil || md.ResolvedFrom != "" || md.GoType != "" {
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
	case md.ResolvedFrom != "":
		return "ref"
	case md.Type == "string" && md.GoType == "time.Time":
		return "datetime"
	case md.Type == "string" && md.GoType == "time.Duration":
		return "duration"
	default:
		return md.Type
	}
}

type DefaultAssigment struct {
	Target     string
	Path       string
	Value      string
	IsPointer  bool
	IsOptional bool
}

func (d *DefaultAssigment) Render() string {
	if d.Target == "" {
		return ""
	}
	// Local variable declaration: "varID := value"
	if d.Path == "" {
		return fmt.Sprintf("\n%s := %s", d.Target, d.Value)
	}
	// Field assignment to cfg (or a nested var): "target.Field = value"
	val := d.Value
	// Pointer and Optional support
	if d.IsPointer {
		val = "&" + val
	}
	if d.IsOptional {
		val = "configoptional.Some(" + val + ")"
	}
	return fmt.Sprintf("%s.%s = %s", d.Target, d.Path, val)
}

// ExtractDefaults returns a flat list of imperative assignment statements for all  properties with defaults, or nil if none exist.
func ExtractDefaults(md *ConfigMetadata) []*DefaultAssigment {
	if md == nil {
		return nil
	}

	stmts := renderObjectFields("cfg", "", md, md.Default)

	if len(stmts) == 0 {
		return nil
	}

	return stmts
}

func renderAssignedDefault(propName, varID, target, fieldPath string, md *ConfigMetadata, defaultValue any) []*DefaultAssigment {
	switch {
	case isObjectWithProperties(md):
		return renderObjectDefault(propName, varID, target, fieldPath, md, defaultValue)

	case isArrayOfObjects(md):
		arr, _ := defaultValue.([]any)
		if len(arr) == 0 {
			return nil
		}
		return renderArrayOfObjectsDefault(propName, varID, target, fieldPath, md, arr)

	case isMapOfObjects(md):
		obj, _ := defaultValue.(map[string]any)
		if len(obj) == 0 {
			return nil
		}
		return renderMapOfObjectsDefault(propName, varID, target, fieldPath, md, obj)

	default:
		return renderSimpleDefault(varID, target, fieldPath, md, defaultValue)
	}
}

func renderObjectDefault(propName, varID, target, fieldPath string, md *ConfigMetadata, defaultValue any) []*DefaultAssigment {
	typeName, err := resolveGoType(md, propName, "", "")
	if err != nil {
		return nil
	}

	itemStmts := renderObjectFields(varID, varID, md, defaultValue)

	if len(itemStmts) == 0 {
		return nil
	}

	stmts := []*DefaultAssigment{{Target: varID, Value: typeName + "{}"}}
	stmts = append(stmts, itemStmts...)
	return append(stmts, newDefaultAssignment(target, fieldPath, varID, md))
}

func renderObjectFields(target, varPrefix string, md *ConfigMetadata, defaultValue any) []*DefaultAssigment {
	overrides, _ := defaultValue.(map[string]any)
	var stmts []*DefaultAssigment

	for _, propName := range slices.Sorted(maps.Keys(md.Properties)) {
		prop := md.Properties[propName]
		fieldPath, err := helpers.FormatIdentifier(propName, true)
		if err != nil {
			continue
		}

		effectiveDefault := prop.Default
		if overrides != nil {
			if v, ok := overrides[propName]; ok {
				effectiveDefault = v
			}
		}

		nestedVarID, ok := makeDefaultVarID(varPrefix, propName)
		if !ok {
			continue
		}

		stmts = append(stmts, renderAssignedDefault(propName, nestedVarID, target, fieldPath, prop, effectiveDefault)...)
	}
	for _, schema := range md.AllOf {
		refTypeName, ok := resolveDefaultRefTypeName(schema)
		if !ok {
			continue
		}

		nestedVarID, ok := makeDefaultVarID(varPrefix, refTypeName)
		if !ok {
			continue
		}

		stmts = append(stmts, renderAssignedDefault(refTypeName, nestedVarID, target, refTypeName, schema, schema.Default)...)
	}

	return stmts
}

func renderArrayOfObjectsDefault(propName, varID, target, fieldPath string, md *ConfigMetadata, arr []any) []*DefaultAssigment {
	itemSchema := md.Items
	itemTypeName, err := resolveGoType(itemSchema, propName+"_item", "", "")
	if err != nil {
		return nil
	}

	var stmts []*DefaultAssigment
	var elemVars []string
	for i, rawItem := range arr {
		elemVarID := fmt.Sprintf("%s_%d", varID, i+1)
		elemStmts := renderObjectFields(elemVarID, elemVarID, itemSchema, rawItem)

		stmts = append(stmts, &DefaultAssigment{Target: elemVarID, Value: itemTypeName + "{}"})
		stmts = append(stmts, elemStmts...)
		elemVars = append(elemVars, elemVarID)
	}

	sliceType, err := resolveGoType(md, propName, "", "")
	if err != nil {
		return nil
	}
	sliceLiteral := sliceType + "{" + strings.Join(elemVars, ", ") + "}"
	stmts = append(stmts, &DefaultAssigment{Target: varID, Value: sliceLiteral})
	return append(stmts, newDefaultAssignment(target, fieldPath, varID, md))
}

func renderMapOfObjectsDefault(propName, varID, target, fieldPath string, md *ConfigMetadata, obj map[string]any) []*DefaultAssigment {
	valueSchema := md.AdditionalProperties
	valueTypeName, err := resolveGoType(valueSchema, propName, "", "")
	if err != nil {
		return nil
	}

	var stmts []*DefaultAssigment
	var mapEntries []string
	for _, key := range slices.Sorted(maps.Keys(obj)) {
		rawValue := obj[key]
		elemVarID := varID + "_" + key
		elemStmts := renderObjectFields(elemVarID, elemVarID, valueSchema, rawValue)

		stmts = append(stmts, &DefaultAssigment{Target: elemVarID, Value: valueTypeName + "{}"})
		stmts = append(stmts, elemStmts...)
		mapEntries = append(mapEntries, fmt.Sprintf(`%q: %s`, key, elemVarID))
	}

	mapType, err := resolveGoType(md, propName, "", "")
	if err != nil {
		return nil
	}
	mapLiteral := mapType + "{" + strings.Join(mapEntries, ", ") + "}"
	stmts = append(stmts, &DefaultAssigment{Target: varID, Value: mapLiteral})
	return append(stmts, newDefaultAssignment(target, fieldPath, varID, md))
}

func renderSimpleDefault(varID, target, fieldPath string, md *ConfigMetadata, defaultValue any) []*DefaultAssigment {
	if defaultValue == nil {
		return nil
	}

	val, ok := renderSimpleValue(md, defaultValue)
	if !ok {
		return nil
	}

	stmts := make([]*DefaultAssigment, 0, 2)
	if md.IsPointer {
		stmts = append(stmts, &DefaultAssigment{Target: varID, Value: val})
		val = varID
	}

	return append(stmts, newDefaultAssignment(target, fieldPath, val, md))
}

func newDefaultAssignment(target, fieldPath, value string, md *ConfigMetadata) *DefaultAssigment {
	return &DefaultAssigment{
		Target:     target,
		Path:       fieldPath,
		Value:      value,
		IsPointer:  md.IsPointer,
		IsOptional: md.IsOptional,
	}
}

func makeDefaultVarID(prefix, name string) (string, bool) {
	if prefix == "" {
		varID, err := helpers.FormatIdentifier(name, false)
		if err != nil {
			return "", false
		}
		return varID, true
	}

	return prefix + "_" + name, true
}

func resolveDefaultRefTypeName(md *ConfigMetadata) (string, bool) {
	if md == nil || md.Default == nil || md.ResolvedFrom == "" {
		return "", false
	}

	ref, err := ResolveGoTypeRef(md.ResolvedFrom, "", "")
	if err != nil {
		return "", false
	}

	return ref.TypeName, true
}

func renderSimpleValue(md *ConfigMetadata, defaultValue any) (string, bool) {
	var goExpr string
	switch v := defaultValue.(type) {
	case string:
		if durationExpr, ok := renderDurationExpr(v); ok {
			goExpr = durationExpr
		} else {
			goExpr = fmt.Sprintf("%q", v)
		}
	case []any:
		// resolve array type
		typeExpr, err := resolveGoType(md, "", "", "")
		if err != nil {
			return "", false
		}
		// map default array
		var values []string
		for _, item := range v {
			if val, ok := renderSimpleValue(md.Items, item); ok {
				values = append(values, val)
			}
		}
		goExpr = typeExpr + "{" + strings.Join(values, " ,") + "}"
	case map[string]any:
		typeExpr, err := resolveGoType(md, "", "", "")
		if err != nil {
			return "", false
		}
		var fields []string
		for _, fieldName := range slices.Sorted(maps.Keys(v)) {
			fieldValue := v[fieldName]
			if val, ok := renderSimpleValue(md.AdditionalProperties, fieldValue); ok {
				fields = append(fields, fmt.Sprintf(`%q: %s`, fieldName, val))
			}
		}
		goExpr = typeExpr + "{" + strings.Join(fields, " ,") + "}"
	default:
		goExpr = fmt.Sprintf("%v", v)
	}
	return goExpr, true
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

func isObjectWithProperties(md *ConfigMetadata) bool {
	return md != nil && md.Type == "object" && len(md.Properties) > 0
}

func isArrayOfObjects(md *ConfigMetadata) bool {
	return md != nil && md.Type == "array" && isObjectWithProperties(md.Items)
}

func isMapOfObjects(md *ConfigMetadata) bool {
	return md != nil && md.Type == "object" && isObjectWithProperties(md.AdditionalProperties)
}
