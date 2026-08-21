// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/schemagen/internal"

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

const (
	defaultFactoryMapKeyVar = "Type"
	factoryConfigTypeName   = "config"
)

// expandFactoryMaps synthesizes typed properties for fields that are excluded from standard
// mapstructure decoding (mapstructure:"-") but whose actual schema is described via a package-level factory map.
func (p *Parser) expandFactoryMaps() error {
	if p.config.Mode != Component || p.config.ComponentOverride == nil {
		return nil
	}
	for _, spec := range p.config.ComponentOverride.FactoryMaps {
		prop, err := p.buildFactoryMapProperty(spec)
		if err != nil {
			return fmt.Errorf("factoryMap %q: %w", spec.Property, err)
		}
		if p.schema.Properties == nil {
			p.schema.Properties = make(map[string]SchemaElement)
		}
		p.schema.Properties[spec.Property] = prop
	}
	return nil
}

// factoryEntry pairs a discriminator key string with the import path of the factory's package.
// When key is empty it must be resolved from the factory package's internal/metadata.
type factoryEntry struct {
	key        string
	importPath string
}

func (p *Parser) buildFactoryMapProperty(spec FactoryMapSpec) (*ObjectSchemaElement, error) {
	entries, err := p.extractFactoryImports(spec.FactoriesVar)
	if err != nil {
		return nil, err
	}

	out := CreateObjectField(spec.Description)
	for _, entry := range entries {
		key := entry.key
		if key == "" {
			key, err = p.resolveDiscriminatorKey(entry.importPath, spec.discriminatorKeyVar())
			if err != nil {
				return nil, fmt.Errorf("factory %q: %w", entry.importPath, err)
			}
		}
		out.AddProperty(key, CreateRefField(p.computeFactoryConfigRefID(entry.importPath), ""))
	}
	return out, nil
}

func (s FactoryMapSpec) discriminatorKeyVar() string {
	if s.KeyFromMetadataVar != "" {
		return s.KeyFromMetadataVar
	}
	return defaultFactoryMapKeyVar
}

// extractFactoryImports walks all files of the loaded package looking for the named var
// and returns factoryEntries extracted from its initializer.
func (p *Parser) extractFactoryImports(varName string) ([]factoryEntry, error) {
	for _, file := range p.pkg.Syntax {
		init, found, err := findInitializer(file, varName, "var", token.VAR)
		if err != nil {
			return nil, err
		}
		if found {
			imports := fileImportMap(file)
			return p.extractEntries(init, imports)
		}
	}
	return nil, fmt.Errorf("var %q not found in package %s", varName, p.pkg.PkgPath)
}

func fileImportMap(file *ast.File) map[string]string {
	imports := make(map[string]string)
	for _, imp := range file.Imports {
		rawPath, alias := ParseImport(imp)
		imports[alias] = rawPath
	}
	return imports
}

func findInitializer(file *ast.File, identName, label string, allowed ...token.Token) (ast.Expr, bool, error) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		if !slices.Contains(allowed, genDecl.Tok) {
			continue
		}
		for _, spec := range genDecl.Specs {
			valSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for index, name := range valSpec.Names {
				if name.Name != identName {
					continue
				}
				if len(valSpec.Values) == 0 {
					return nil, true, fmt.Errorf("%s %q has no initializer", label, identName)
				}
				if len(valSpec.Values) == 1 {
					return valSpec.Values[0], true, nil
				}
				if index < len(valSpec.Values) {
					return valSpec.Values[index], true, nil
				}
				return nil, true, fmt.Errorf("%s %q has no initializer", label, identName)
			}
		}
	}
	return nil, false, nil
}

// extractEntries dispatches on the initializer shape:
//   - CallExpr:        someFunc(pkg.NewFactory(), ...)  — key resolved later from metadata
//   - CompositeLit:    map[T]F{key: pkg.NewFactory(), ...} — key inline or qualified
func (p *Parser) extractEntries(expr ast.Expr, fileImports map[string]string) ([]factoryEntry, error) {
	switch v := expr.(type) {
	case *ast.CallExpr:
		return extractCallArgEntries(v.Args, fileImports)
	case *ast.CompositeLit:
		return p.extractCompositeLitEntries(v, fileImports)
	default:
		return nil, fmt.Errorf("unsupported factory var initializer type %T", expr)
	}
}

func extractCallArgEntries(args []ast.Expr, fileImports map[string]string) ([]factoryEntry, error) {
	var entries []factoryEntry
	for _, arg := range args {
		importPath, err := extractPkgFromFactoryCall(arg, fileImports)
		if err != nil {
			return nil, err
		}
		entries = append(entries, factoryEntry{importPath: importPath})
	}
	return entries, nil
}

func (p *Parser) extractCompositeLitEntries(lit *ast.CompositeLit, fileImports map[string]string) ([]factoryEntry, error) {
	var entries []factoryEntry
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return nil, fmt.Errorf("expected key-value pair in composite literal, got %T", elt)
		}
		key, err := p.resolveKeyExpr(kv.Key, fileImports)
		if err != nil {
			return nil, err
		}
		importPath, err := extractPkgFromFactoryCall(kv.Value, fileImports)
		if err != nil {
			return nil, err
		}
		entries = append(entries, factoryEntry{key: key, importPath: importPath})
	}
	return entries, nil
}

// resolveKeyExpr extracts the discriminator key string from a map-key expression.
// It handles three shapes:
//   - *ast.BasicLit (string):  "key"
//   - *ast.CallExpr:           component.MustNewType("key")
//   - *ast.SelectorExpr:       pkg.ConstOrVar  (resolved by loading the package)
func (p *Parser) resolveKeyExpr(expr ast.Expr, fileImports map[string]string) (string, error) {
	switch v := expr.(type) {
	case *ast.BasicLit:
		return stringLiteral(v, "map key")
	case *ast.CallExpr:
		return singleStringArg(v, "key call")
	case *ast.SelectorExpr:
		pkgIdent, ok := v.X.(*ast.Ident)
		if !ok {
			return "", fmt.Errorf("expected ident in selector key, got %T", v.X)
		}
		importPath, found := fileImports[pkgIdent.Name]
		if !found {
			return "", fmt.Errorf("import %q not found in file imports", pkgIdent.Name)
		}
		return p.resolvePackageStringValue(importPath, v.Sel.Name)
	default:
		return "", fmt.Errorf("unsupported map key expression type %T", expr)
	}
}

// extractPkgFromFactoryCall resolves the import path of a factory expression.
// Handles: pkg.NewFactory(), &pkg.Factory{}, pkg.Factory{}.
func extractPkgFromFactoryCall(expr ast.Expr, fileImports map[string]string) (string, error) {
	if unary, ok := expr.(*ast.UnaryExpr); ok {
		expr = unary.X
	}
	var sel *ast.SelectorExpr
	switch v := expr.(type) {
	case *ast.CallExpr:
		sel, _ = v.Fun.(*ast.SelectorExpr)
	case *ast.CompositeLit:
		sel, _ = v.Type.(*ast.SelectorExpr)
	default:
		return "", fmt.Errorf("expected factory call or literal, got %T", expr)
	}
	if sel == nil {
		return "", errors.New("expected selector expression in factory expression")
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", fmt.Errorf("expected ident in factory selector, got %T", sel.X)
	}
	importPath, found := fileImports[pkgIdent.Name]
	if !found {
		return "", fmt.Errorf("import %q not found in file imports", pkgIdent.Name)
	}
	return importPath, nil
}

func (p *Parser) resolveDiscriminatorKey(importPath, keyVarName string) (string, error) {
	return p.resolvePackageStringValue(importPath+"/internal/metadata", keyVarName)
}

func (p *Parser) resolvePackageStringValue(importPath, identName string) (string, error) {
	set := token.NewFileSet()
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax,
		Fset: set,
		Dir:  p.config.DirPath,
	}, importPath)
	if err != nil {
		return "", fmt.Errorf("load package %s: %w", importPath, err)
	}
	if len(pkgs) == 0 {
		return "", fmt.Errorf("no packages found for %s", importPath)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			val, found, err := findInitializer(file, identName, "identifier", token.VAR, token.CONST)
			if err != nil {
				return "", err
			}
			if found {
				return stringValue(identName, val)
			}
		}
	}
	return "", fmt.Errorf("identifier %q not found in package %s", identName, importPath)
}

func stringValue(identName string, expr ast.Expr) (string, error) {
	switch v := expr.(type) {
	case *ast.BasicLit:
		return stringLiteral(v, fmt.Sprintf("identifier %q", identName))
	case *ast.CallExpr:
		return singleStringArg(v, fmt.Sprintf("identifier %q call", identName))
	default:
		return "", fmt.Errorf("identifier %q has unsupported initializer type %T", identName, expr)
	}
}

func stringLiteral(lit *ast.BasicLit, context string) (string, error) {
	if lit.Kind != token.STRING {
		return "", fmt.Errorf("expected string literal as %s, got %s", context, lit.Kind)
	}
	return strconv.Unquote(lit.Value)
}

func singleStringArg(callExpr *ast.CallExpr, context string) (string, error) {
	if len(callExpr.Args) != 1 {
		return "", fmt.Errorf("expected 1 arg in %s, got %d", context, len(callExpr.Args))
	}
	lit, ok := callExpr.Args[0].(*ast.BasicLit)
	if !ok {
		return "", fmt.Errorf("expected string literal as %s arg, got %T", context, callExpr.Args[0])
	}
	return stringLiteral(lit, context+" arg")
}

func (p *Parser) computeFactoryConfigRefID(importPath string) string {
	if strings.HasPrefix(importPath, p.pkg.PkgPath+"/") {
		relPath := "." + strings.TrimPrefix(importPath, p.pkg.PkgPath)
		return relPath + "." + factoryConfigTypeName
	}

	fullID := importPath + "." + factoryConfigTypeName
	if p.config.Namespace != "" && (strings.HasPrefix(importPath, p.config.Namespace+"/") || importPath == p.config.Namespace) {
		refID, _ := strings.CutPrefix(fullID, p.config.Namespace)
		return refID
	}

	return fullID
}
