// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/schemagen"

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/packages"
)

// PackageAnalyzer analyzes Go packages to extract struct information.
type PackageAnalyzer struct {
	dir string
	// pkgCache caches loaded packages to avoid reloading them
	pkgCache map[string]*packages.Package
}

// NewPackageAnalyzer creates a new PackageAnalyzer for the given directory.
func NewPackageAnalyzer(dir string) *PackageAnalyzer {
	return &PackageAnalyzer{
		dir:      dir,
		pkgCache: make(map[string]*packages.Package),
	}
}

// analyzeConfig loads the package and finds the Config struct.
// configTypeName is the name of the config type (e.g., "Config"). If empty, auto-detection is used.
// configPkgPath is the package path where the config is defined. If empty, the local package is used.
func (a *PackageAnalyzer) analyzeConfig(configTypeName, configPkgPath string) (*StructInfo, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedSyntax | packages.NeedImports | packages.NeedDeps,
		Dir: a.dir,
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found in %s", a.dir)
	}

	if len(pkgs[0].Errors) > 0 {
		var errMsgs []string
		for _, e := range pkgs[0].Errors {
			errMsgs = append(errMsgs, e.Error())
		}
		return nil, fmt.Errorf("package errors: %s", strings.Join(errMsgs, "; "))
	}

	pkg := pkgs[0]

	// Auto-detect config type if not specified
	if configTypeName == "" {
		configTypeName = a.detectConfigFromVarDecl(pkg)
		if configTypeName == "" {
			configTypeName = "Config" // fallback to default
		}
	}

	// Determine which package to analyze
	analyzePkg := pkg
	if configPkgPath != "" {
		// Config is in an external package
		analyzePkg, err = a.loadPackage(configPkgPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config package %s: %w", configPkgPath, err)
		}
	}

	// Find the Config type
	obj := analyzePkg.Types.Scope().Lookup(configTypeName)
	if obj == nil {
		return nil, fmt.Errorf("type %s not found in package %s", configTypeName, analyzePkg.PkgPath)
	}

	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil, fmt.Errorf("%s is not a named type", configTypeName)
	}

	structType, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("%s is not a struct type", configTypeName)
	}

	// Extract fields
	fields := a.extractFields(analyzePkg, structType)

	return &StructInfo{
		Name:    configTypeName,
		Package: analyzePkg.PkgPath,
		Fields:  fields,
	}, nil
}

// extractFields extracts field information from a struct type.
func (a *PackageAnalyzer) extractFields(pkg *packages.Package, st *types.Struct) []FieldInfo {
	var fields []FieldInfo

	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		tag := st.Tag(i)

		fieldInfo := a.extractFieldInfo(pkg, field, tag)
		if fieldInfo != nil {
			fields = append(fields, *fieldInfo)
		}
	}

	return fields
}

// extractFieldInfo extracts information about a single field.
func (a *PackageAnalyzer) extractFieldInfo(pkg *packages.Package, field *types.Var, tag string) *FieldInfo {
	// Skip unexported fields
	if !field.Exported() {
		return nil
	}

	// Parse struct tags
	jsonName, squash := parseTag(tag)
	if jsonName == "-" {
		return nil // Skip fields with json:"-" or mapstructure:"-"
	}
	if jsonName == "" {
		jsonName = strings.ToLower(field.Name())
	}

	// Check if embedded - either Go-level embedding or mapstructure squash
	embedded := field.Embedded() || squash

	// Get field doc comment
	fieldDoc := a.findFieldDoc(pkg, field)

	typeStr := resolveTypeAlias(field.Type())

	info := &FieldInfo{
		Name:        field.Name(),
		JSONName:    jsonName,
		Type:        typeStr,
		Description: fieldDoc,
		Embedded:    embedded,
	}

	// For embedded structs or struct fields, extract nested fields
	if st := getUnderlyingStruct(field.Type()); st != nil {
		info.Fields = a.extractFields(pkg, st)
	}

	return info
}

// findFieldDoc finds the documentation comment for a struct field.
// It searches in the field's originating package, loading external packages if needed.
func (a *PackageAnalyzer) findFieldDoc(pkg *packages.Package, field *types.Var) string {
	pos := field.Pos()
	if !pos.IsValid() {
		return ""
	}

	// Determine which package the field comes from
	fieldPkg := field.Pkg()
	if fieldPkg == nil {
		return ""
	}

	// Get the package to search in - either the local package or load the external one
	searchPkg := pkg
	if fieldPkg.Path() != pkg.PkgPath {
		// Field comes from an external package, need to load it
		var err error
		searchPkg, err = a.loadPackage(fieldPkg.Path())
		if err != nil || searchPkg == nil {
			return ""
		}
	}

	return a.findFieldDocInPackage(searchPkg, field)
}

// loadPackage loads a package by its import path, using the cache if available.
func (a *PackageAnalyzer) loadPackage(pkgPath string) (*packages.Package, error) {
	// Check cache first
	if cached, ok := a.pkgCache[pkgPath]; ok {
		return cached, nil
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedSyntax | packages.NeedImports | packages.NeedDeps,
		Dir: a.dir,
	}

	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil, err
	}

	if len(pkgs) == 0 {
		return nil, nil
	}

	// Cache the loaded package
	a.pkgCache[pkgPath] = pkgs[0]
	return pkgs[0], nil
}

// findFieldDocInPackage searches for a field's documentation in a specific package's AST.
func (a *PackageAnalyzer) findFieldDocInPackage(pkg *packages.Package, field *types.Var) string {
	if pkg == nil || pkg.Syntax == nil {
		return ""
	}

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, f := range st.Fields.List {
					for _, name := range f.Names {
						if name.Name == field.Name() {
							if f.Doc != nil {
								return strings.TrimSpace(f.Doc.Text())
							}
							if f.Comment != nil {
								return strings.TrimSpace(f.Comment.Text())
							}
							return ""
						}
					}
				}
			}
		}
	}
	return ""
}

// parseTag parses struct tags to find the mapstructure name and squash option.
func parseTag(tag string) (name string, squash bool) {
	st := reflect.StructTag(tag)

	if ms := st.Get("mapstructure"); ms != "" {
		parts := strings.Split(ms, ",")
		name = parts[0]
		for _, p := range parts[1:] {
			if p == "squash" {
				squash = true
			}
		}
		return name, squash
	}

	return "", false
}

// detectConfigFromVarDecl looks for the component.Config interface assignment pattern:
// var _ component.Config = (*TypeName)(nil)
func (a *PackageAnalyzer) detectConfigFromVarDecl(pkg *packages.Package) string {
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			for _, spec := range genDecl.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}

				// Check if this is a blank identifier assignment: var _ Type = ...
				if len(valueSpec.Names) != 1 || valueSpec.Names[0].Name != "_" {
					continue
				}

				// Check if the type is component.Config
				if !isComponentConfigType(valueSpec.Type) {
					continue
				}

				// Extract the type from the value: (*TypeName)(nil)
				if len(valueSpec.Values) != 1 {
					continue
				}

				typeName := extractPointerTypeName(valueSpec.Values[0])
				if typeName != "" {
					return typeName
				}
			}
		}
	}
	return ""
}

// isComponentConfigType checks if the type expression is component.Config.
func isComponentConfigType(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == "component" && sel.Sel.Name == "Config"
}

// extractPointerTypeName extracts TypeName from (*TypeName)(nil).
func extractPointerTypeName(expr ast.Expr) string {
	// Pattern: (*TypeName)(nil) is a CallExpr
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return ""
	}

	// The function part should be (*TypeName)
	paren, ok := call.Fun.(*ast.ParenExpr)
	if !ok {
		return ""
	}

	// Inside parens should be *TypeName
	star, ok := paren.X.(*ast.StarExpr)
	if !ok {
		return ""
	}

	// Extract the type name
	ident, ok := star.X.(*ast.Ident)
	if !ok {
		return ""
	}

	return ident.Name
}

// getUnderlyingStruct returns the underlying struct type if t is a struct or pointer to struct.
func getUnderlyingStruct(t types.Type) *types.Struct {
	// Handle pointer types
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	// Handle slice types - extract element type
	if slice, ok := t.(*types.Slice); ok {
		return getUnderlyingStruct(slice.Elem())
	}

	// Handle named types (including generics like configoptional.Optional[T])
	if named, ok := t.(*types.Named); ok {
		// Check if this is an Optional[T] generic and unwrap T
		if isOptionalType(t) {
			typeArgs := named.TypeArgs()
			// Recursively get the struct from the inner type
			return getUnderlyingStruct(typeArgs.At(0))
		}
		t = named.Underlying()
	}

	if st, ok := t.(*types.Struct); ok {
		return st
	}
	return nil
}

// isOptionalType checks if a type is a generic Optional type (e.g., configoptional.Optional[T]).
// Uses type structure rather than string matching for robustness.
func isOptionalType(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}

	// Check if it's a generic type with type arguments
	if named.TypeArgs() == nil || named.TypeArgs().Len() == 0 {
		return false
	}

	// Check the type name (without package path)
	return named.Obj().Name() == "Optional"
}

// resolveTypeAlias resolves type aliases to their underlying basic types.
// For example, "type CustomString string" returns "string".
// Preserves well-known types like time.Duration that need special handling.
// For complex types (structs, etc.), returns the original type string.
func resolveTypeAlias(t types.Type) string {
	// If it's a named type, check if it's an alias for a basic type
	if named, ok := t.(*types.Named); ok {
		typeName := named.Obj().Name()
		pkgPath := ""
		if named.Obj().Pkg() != nil {
			pkgPath = named.Obj().Pkg().Path()
		}

		// Preserve well-known types that need special handling
		switch {
		case pkgPath == "time" && (typeName == "Duration" || typeName == "Time"):
			return t.String()
		case typeName == "ID" || typeName == "Type":
			// component.ID and component.Type
			return t.String()
		case strings.Contains(t.String(), "configopaque") || strings.Contains(t.String(), "configoptional"):
			// Preserve config types for special handling
			return t.String()
		}

		underlying := named.Underlying()
		// If the underlying type is a basic type, use that
		if basic, ok := underlying.(*types.Basic); ok {
			return basic.String()
		}
	}
	// For all other cases, return the original type string
	return t.String()
}
