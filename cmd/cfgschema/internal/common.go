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

package internal

import (
	"path"
	"reflect"
	"strings"
)

const DefaultSrcRoot = "."
const DefaultModule = "go.opentelemetry.io/collector"

type Env struct {
	SrcRoot          string
	ModuleName       string
	GetTargetYamlDir func(reflect.Type, Env) string
}

func PackageDir(t reflect.Type, env Env) string {
	pkg := strings.TrimPrefix(t.PkgPath(), env.ModuleName+"/")
	return path.Join(env.SrcRoot, pkg)
}
