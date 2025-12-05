// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/cmd/builder/internal/schemagen"

import (
	"fmt"
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
		return nil, fmt.Errorf("package %s has errors: %v", importPath, pkg.Errors[0])
	}

	pa.packages[importPath] = pkg
	return pkg, nil
}

// FindConfigType locates the Config struct in a component package.
// It looks for a type named "Config" that is a struct.
func (pa *PackageAnalyzer) FindConfigType(pkg *packages.Package) (*types.Named, error) {
	if pkg.Types == nil {
		return nil, fmt.Errorf("package %s has no type information", pkg.PkgPath)
	}

	scope := pkg.Types.Scope()

	// Look for "Config" type
	obj := scope.Lookup("Config")
	if obj == nil {
		return nil, fmt.Errorf("no Config type found in package %s", pkg.PkgPath)
	}

	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil, fmt.Errorf("Config in package %s is not a named type", pkg.PkgPath)
	}

	// Verify it's a struct
	if _, ok := named.Underlying().(*types.Struct); !ok {
		return nil, fmt.Errorf("Config in package %s is not a struct", pkg.PkgPath)
	}

	return named, nil
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
