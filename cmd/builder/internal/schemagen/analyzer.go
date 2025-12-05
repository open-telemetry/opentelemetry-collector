// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/cmd/builder/internal/schemagen"

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// PackageAnalyzer loads Go packages and extracts type information using static analysis.
type PackageAnalyzer struct {
	packages map[string]*packages.Package
	cfg      *packages.Config
}

// NewPackageAnalyzer creates a new package analyzer.
// The workDir should be the directory containing the go.mod file of the generated collector.
func NewPackageAnalyzer(workDir string) *PackageAnalyzer {
	return &PackageAnalyzer{
		packages: make(map[string]*packages.Package),
		cfg: &packages.Config{
			Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax |
				packages.NeedTypesInfo | packages.NeedImports,
			Dir: workDir,
		},
	}
}

// LoadPackage loads a Go package by import path and returns it.
// Results are cached for efficiency.
func (pa *PackageAnalyzer) LoadPackage(importPath string) (*packages.Package, error) {
	if pkg, exists := pa.packages[importPath]; exists {
		return pkg, nil
	}

	pkgs, err := packages.Load(pa.cfg, importPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load package %s: %w", importPath, err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no package found for import path: %s", importPath)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package %s has errors: %w", importPath, pkg.Errors[0])
	}

	pa.packages[importPath] = pkg
	return pkg, nil
}

// FindConfigType locates the Config struct in a component package.
// It first searches for compile-time check patterns like:
//
//	var _ component.Config = (*ConfigType)(nil)
//
// If no such pattern is found, it falls back to looking for a type named "Config".
func (pa *PackageAnalyzer) FindConfigType(pkg *packages.Package) (*types.Named, error) {
	if pkg.Types == nil {
		return nil, fmt.Errorf("package %s has no type information", pkg.PkgPath)
	}

	// Search AST for compile-time check pattern
	configTypeName := pa.findConfigTypeFromAST(pkg)
	if configTypeName != "" {
		return pa.findConfigTypeByName(pkg, configTypeName)
	}

	// Fallback to looking for "Config" type by name
	return pa.findConfigTypeByName(pkg, "Config")
}

// findConfigTypeByName looks up a type by name in the package scope.
func (pa *PackageAnalyzer) findConfigTypeByName(pkg *packages.Package, typeName string) (*types.Named, error) {
	scope := pkg.Types.Scope()

	obj := scope.Lookup(typeName)
	if obj == nil {
		return nil, fmt.Errorf("no %s type found in package %s", typeName, pkg.PkgPath)
	}

	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil, fmt.Errorf("%s in package %s is not a named type", typeName, pkg.PkgPath)
	}

	// Verify it's a struct
	if _, ok := named.Underlying().(*types.Struct); !ok {
		return nil, fmt.Errorf("%s in package %s is not a struct", typeName, pkg.PkgPath)
	}

	return named, nil
}

// findConfigTypeFromAST searches for compile-time check patterns in the package AST.
// It looks for patterns like: var _ component.Config = (*ConfigType)(nil)
func (pa *PackageAnalyzer) findConfigTypeFromAST(pkg *packages.Package) string {
	for _, file := range pkg.Syntax {
		if typeName := pa.findConfigTypeInFile(file); typeName != "" {
			return typeName
		}
	}
	return ""
}

// findConfigTypeInFile searches a single file for the compile-time check pattern.
func (pa *PackageAnalyzer) findConfigTypeInFile(file *ast.File) string {
	// Build a map of local import names to their full import paths
	importAliases := buildImportAliasMap(file)

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			if typeName := pa.extractConfigTypeFromValueSpec(valueSpec, importAliases); typeName != "" {
				return typeName
			}
		}
	}
	return ""
}

// buildImportAliasMap creates a map from local package names to their import paths.
// For example, if a file has:
//
//	import comp "go.opentelemetry.io/collector/component"
//
// The map will contain: {"comp": "go.opentelemetry.io/collector/component"}
func buildImportAliasMap(file *ast.File) map[string]string {
	aliases := make(map[string]string)
	for _, imp := range file.Imports {
		// Get the import path (remove quotes)
		importPath := imp.Path.Value[1 : len(imp.Path.Value)-1]

		var localName string
		if imp.Name != nil {
			// Explicit alias: import foo "path/to/pkg"
			localName = imp.Name.Name
		} else {
			// Default: use the last segment of the import path
			localName = importPathToName(importPath)
		}

		aliases[localName] = importPath
	}
	return aliases
}

// importPathToName extracts the package name from an import path.
// For "go.opentelemetry.io/collector/component" it returns "component".
func importPathToName(importPath string) string {
	for i := len(importPath) - 1; i >= 0; i-- {
		if importPath[i] == '/' {
			return importPath[i+1:]
		}
	}
	return importPath
}

// extractConfigTypeFromValueSpec checks if a ValueSpec matches the pattern:
// var _ component.Config = (*ConfigType)(nil)
func (pa *PackageAnalyzer) extractConfigTypeFromValueSpec(spec *ast.ValueSpec, importAliases map[string]string) string {
	// Must have exactly one name and it must be blank identifier
	if len(spec.Names) != 1 || spec.Names[0].Name != "_" {
		return ""
	}

	// Check if the type is component.Config
	if !isComponentConfigType(spec.Type, importAliases) {
		return ""
	}

	// Must have exactly one value
	if len(spec.Values) != 1 {
		return ""
	}

	// Extract type name from (*TypeName)(nil)
	return pa.extractTypeFromConversionExpr(spec.Values[0])
}

// componentImportPath is the canonical import path for the component package.
const componentImportPath = "go.opentelemetry.io/collector/component"

// isComponentConfigType checks if the type expression represents component.Config.
// It uses the importAliases map to resolve aliased imports.
func isComponentConfigType(typeExpr ast.Expr, importAliases map[string]string) bool {
	sel, ok := typeExpr.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Check for "Config" selector
	if sel.Sel.Name != "Config" {
		return false
	}

	// Check for package identifier
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	// Look up the import path for this identifier
	importPath, exists := importAliases[ident.Name]
	if !exists {
		return false
	}

	// Check if it's the component package
	return importPath == componentImportPath
}

// extractTypeFromConversionExpr extracts the type name from a (*TypeName)(nil) expression.
func (pa *PackageAnalyzer) extractTypeFromConversionExpr(expr ast.Expr) string {
	// The expression should be a call expression: (*TypeName)(nil)
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		return ""
	}

	// Should have exactly one argument (nil)
	if len(callExpr.Args) != 1 {
		return ""
	}

	// Verify the argument is nil
	ident, ok := callExpr.Args[0].(*ast.Ident)
	if !ok || ident.Name != "nil" {
		return ""
	}

	// Fun should be a parenthesized star expression: (*TypeName)
	parenExpr, ok := callExpr.Fun.(*ast.ParenExpr)
	if !ok {
		return ""
	}

	starExpr, ok := parenExpr.X.(*ast.StarExpr)
	if !ok {
		return ""
	}

	// Extract the type name
	typeIdent, ok := starExpr.X.(*ast.Ident)
	if !ok {
		return ""
	}

	return typeIdent.Name
}

// GetStructFromNamed extracts the underlying struct type from a named type.
func GetStructFromNamed(named *types.Named) (*types.Struct, bool) {
	st, ok := named.Underlying().(*types.Struct)
	return st, ok
}

// GetPackage returns a cached package by import path, or nil if not loaded.
func (pa *PackageAnalyzer) GetPackage(importPath string) *packages.Package {
	return pa.packages[importPath]
}
