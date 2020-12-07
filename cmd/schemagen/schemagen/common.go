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

package schemagen

import (
	"path"
	"reflect"
	"strings"
)

const defaultSrcRoot = "."
const defaultModule = "go.opentelemetry.io/collector"

type env struct {
	srcRoot      string
	moduleName   string
	yamlFilename func(reflect.Type, env) string
}

const schemaFilename = "cfg-schema.yaml"

func yamlFilename(t reflect.Type, env env) string {
	return path.Join(packageDir(t, env), schemaFilename)
}

func packageDir(t reflect.Type, env env) string {
	pkg := strings.TrimPrefix(t.PkgPath(), env.moduleName+"/")
	return path.Join(env.srcRoot, pkg)
}
