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

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
)

// fieldCommentsForStruct returns a map of fieldname -> comment for a struct
func fieldCommentsForStruct(v reflect.Value) map[string]string {
	elem := v
	if v.Kind() == reflect.Ptr {
		elem = v.Elem()
	}
	path := typePath(elem.Type())
	name := trimPackage(elem)
	return fieldCommentsForName(path, name)
}

func typePath(t reflect.Type) string {
	const moduleName = "go.opentelemetry.io/collector/"
	return strings.TrimPrefix(t.PkgPath(), moduleName)
}

func trimPackage(elem reflect.Value) string {
	typeName := elem.Type().String()
	split := strings.Split(typeName, ".")
	return split[1]
}

func fieldCommentsForName(packageDir, structName string) map[string]string {
	configStruct := parseDir(packageDir, structName)
	if configStruct == nil {
		return nil
	}
	fieldComments := map[string]string{}
	if structType, ok := configStruct.Type.(*ast.StructType); ok {
		for _, field := range structType.Fields.List {
			key := ""
			if field.Names != nil {
				key = field.Names[0].Name
			} else if typ, ok := field.Type.(*ast.SelectorExpr); ok {
				key = typ.Sel.Name
			} else if typ, ok := field.Type.(*ast.Ident); ok {
				key = typ.Name
			} else {
				continue
			}
			fieldComments[key] = field.Doc.Text()
		}
	}
	return fieldComments
}

func parseDir(packageDir, structName string) *ast.TypeSpec {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, packageDir, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			cmap := ast.NewCommentMap(fset, file, file.Comments)
			for node := range cmap {
				if t, ok := node.(*ast.GenDecl); ok {
					if t.Tok != token.TYPE {
						continue
					}
					spec := t.Specs[0] // i've only ever seen len(t.Specs) == 1
					if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Name.Name == structName {
						return typeSpec
					}
				}
			}
		}
	}
	return nil
}
