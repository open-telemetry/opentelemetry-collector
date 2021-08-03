// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configschema

import (
	"go/build"
	"os"
	"path"
	"reflect"
	"strings"
)

// DefaultSrcRoot is the default root of the collector repo, relative to the
// current working directory. Can be used to create a DirResolver.
const DefaultSrcRoot = "."

// DefaultModule is the module prefix of otelcol. Can be used to create a
// DirResolver.
const DefaultModule = "go.opentelemetry.io/collector"

// DirResolver is used to resolve the base directory of a given reflect.Type.
type DirResolver struct {
	SrcRoot    string
	ModuleName string
}

// NewDefaultDirResolver creates a DirResolver with a default SrcRoot and
// ModuleName, suitable for using this package's API using otelcol with an
// executable running from the otelcol's source root (not tests).
func NewDefaultDirResolver() DirResolver {
	return NewDirResolver(DefaultSrcRoot, DefaultModule)
}

// NewDirResolver creates a DirResolver with a custom SrcRoot and ModuleName.
// Useful for testing and for using this package's API from a repository other
// than otelcol (e.g. contrib).
func NewDirResolver(srcRoot string, moduleName string) DirResolver {
	return DirResolver{
		SrcRoot:    srcRoot,
		ModuleName: moduleName,
	}
}

// PackageDir accepts a Type and returns its package dir.
func (dr DirResolver) PackageDir(t reflect.Type) string {
	pkg := strings.TrimPrefix(t.PkgPath(), dr.ModuleName+"/")
	dir := localPackageDir(dr, pkg)
	if _, err := os.ReadDir(dir); err != nil {
		dir = externalPackageDir(dr, pkg)
	}
	return dir
}

// localPackageDir returns the path to a local package.
func localPackageDir(dr DirResolver, pkg string) string {
	return path.Join(dr.SrcRoot, pkg)
}

// externalPackageDir returns the path to an external package.
func externalPackageDir(dr DirResolver, pkg string) string {
	line := grepMod(path.Join(dr.SrcRoot, "go.mod"), pkg)
	v := extractVersion(line)
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		goPath = build.Default.GOPATH
	}
	pkgPath := buildExternalPath(goPath, pkg, v)
	if _, err := os.ReadDir(pkgPath); err != nil {
		panic(err)
	}
	return pkgPath
}

// grepMod returns the line within go.mod associated with the package
// we are looking for.
func grepMod(goModPath string, pkg string) string {
	file, _ := os.ReadFile(goModPath)
	goModFile := strings.Split(string(file), "\n")
	var pkgPath string
	for _, line := range goModFile {
		if strings.Contains(line, pkg) || strings.Contains(line, DefaultModule+" ") ||
			strings.Contains(line, "replace") && strings.Contains(line, pkg) {
			pkgPath = line
		}
	}
	return pkgPath
}

// extractVersion gives us the version of the package from go.mod.
func extractVersion(line string) string {
	split := strings.Split(line, " ")
	return split[len(split)-1]
}

// buildExternalPath builds a path to a package that is not local to directory.
func buildExternalPath(goPath, pkg, v string) string {
	switch {
	case strings.HasPrefix(v, "./"):
		return v
	case strings.HasPrefix(v, "../"):
		return path.Join(v, strings.TrimPrefix(pkg, DefaultModule))
	default:
		return path.Join(goPath, "pkg", "mod", pkg+"@"+v)
	}
}
