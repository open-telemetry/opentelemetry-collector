// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/helpers"
)

// GoTypeRef represents a fully resolved Go type reference for code generation.
// It holds the import path and exported type name needed to render a type in generated Go source code.
type GoTypeRef struct {
	// ImportPath is the full Go import path
	// Empty for internal (local $defs) references that need no import.
	ImportPath string
	// TypeName is the exported Go type name
	TypeName string
}

// Qualifier returns the short package name used as a qualifier in Go source.
// Returns "" for internal references (no import needed).
func (r GoTypeRef) Qualifier() string {
	if r.ImportPath == "" {
		return ""
	}
	return path.Base(r.ImportPath)
}

// String returns the Go type expression as it would appear in source code,
func (r GoTypeRef) String() string {
	if q := r.Qualifier(); q != "" {
		return q + "." + r.TypeName
	}
	return r.TypeName
}

// ResolveGoTypeRef converts a raw metadata reference string into a GoTypeRef.
//
// Parameters:
//   - ref:              raw reference string from metadata
//   - rootPackage:      module path from the repo-root go.mod
//   - componentPackage: full Go import path of the component
func ResolveGoTypeRef(ref, rootPackage, componentPackage string) (GoTypeRef, error) {
	if ref == "" {
		return GoTypeRef{}, errors.New("empty reference string")
	}

	cleanRef, _, _ := strings.Cut(ref, "@")

	switch {
	case strings.HasPrefix(cleanRef, "/"):
		return resolveLocalAbsolute(cleanRef, rootPackage)
	case strings.HasPrefix(cleanRef, "./") || strings.HasPrefix(cleanRef, "../"):
		return resolveLocalRelative(cleanRef, componentPackage)
	case strings.Contains(cleanRef, "/"):
		return resolveExternal(cleanRef)
	default:
		return resolveInternal(cleanRef)
	}
}

func resolveInternal(ref string) (GoTypeRef, error) {
	typeName, err := helpers.FormatIdentifier(ref, true)
	if err != nil {
		return GoTypeRef{}, fmt.Errorf("failed to format internal type %q: %w", ref, err)
	}
	return GoTypeRef{ImportPath: "", TypeName: typeName}, nil
}

func resolveExternal(ref string) (GoTypeRef, error) {
	sepIndex := strings.LastIndex(ref, ".")
	if sepIndex == -1 || sepIndex == len(ref)-1 {
		return GoTypeRef{}, fmt.Errorf("invalid external reference %q: missing type name after last dot", ref)
	}
	pkgPath := ref[:sepIndex]
	rawType := ref[sepIndex+1:]

	typeName, err := helpers.FormatIdentifier(rawType, true)
	if err != nil {
		return GoTypeRef{}, fmt.Errorf("failed to format external type %q: %w", rawType, err)
	}
	return GoTypeRef{ImportPath: pkgPath, TypeName: typeName}, nil
}

func resolveLocalAbsolute(ref, rootPackage string) (GoTypeRef, error) {
	sepIndex := strings.LastIndex(ref, ".")
	if sepIndex == -1 || sepIndex == len(ref)-1 {
		return GoTypeRef{}, fmt.Errorf("invalid local absolute reference %q: missing type name after last dot", ref)
	}
	localPath := ref[:sepIndex]
	rawType := ref[sepIndex+1:]

	typeName, err := helpers.FormatIdentifier(rawType, true)
	if err != nil {
		return GoTypeRef{}, fmt.Errorf("failed to format local type %q: %w", rawType, err)
	}

	importPath := rootPackage + localPath
	return GoTypeRef{ImportPath: importPath, TypeName: typeName}, nil
}

func resolveLocalRelative(ref, componentPackage string) (GoTypeRef, error) {
	sepIndex := strings.LastIndex(ref, ".")
	if sepIndex == -1 || sepIndex == len(ref)-1 {
		return GoTypeRef{}, fmt.Errorf("invalid local relative reference %q: missing type name after last dot", ref)
	}
	relPath := ref[:sepIndex]
	rawType := ref[sepIndex+1:]

	typeName, err := helpers.FormatIdentifier(rawType, true)
	if err != nil {
		return GoTypeRef{}, fmt.Errorf("failed to format local type %q: %w", rawType, err)
	}

	importPath := path.Join(componentPackage, relPath)
	return GoTypeRef{ImportPath: importPath, TypeName: typeName}, nil
}
