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
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
)

// commentsForStruct returns a map of fieldname -> comment for a struct
func commentsForStruct(v reflect.Value, dr DirResolver) (map[string]string, error) {
	elem := v
	if v.Kind() == reflect.Ptr {
		elem = v.Elem()
	}
	dir, err := dr.PackageDir(elem.Type())
	if err != nil {
		return nil, err
	}
	name := trimPackage(elem)
	return commentsForStructName(dir, name), nil
}

func trimPackage(v reflect.Value) string {
	typeName := v.Type().String()
	split := strings.Split(typeName, ".")
	return split[1]
}

func commentsForStructName(packageDir, structName string) map[string]string {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, packageDir, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	comments := map[string]string{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				if gd, ok := decl.(*ast.GenDecl); ok {
					for _, spec := range gd.Specs {
						if ts, ok := spec.(*ast.TypeSpec); ok {
							if ts.Name.Name == structName {
								if structComments := gd.Doc.Text(); structComments != "" {
									comments["_struct"] = structComments
								}
								if st, ok := ts.Type.(*ast.StructType); ok {
									for _, field := range st.Fields.List {
										if name := fieldName(field); name != "" {
											comments[name] = field.Doc.Text()
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return comments
}

func fieldName(field *ast.Field) string {
	if field.Names != nil {
		return field.Names[0].Name
	} else if se, ok := field.Type.(*ast.SelectorExpr); ok {
		return se.Sel.Name
	} else if id, ok := field.Type.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}
