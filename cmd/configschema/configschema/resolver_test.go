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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/model/pdata"
)

func TestExternalPkgDir(t *testing.T) {
	pkgPath, _ := externalPackageDir(testDR(), "grpc-ecosystem/grpc-gateway")
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		goPath = build.Default.GOPATH
	}
	assert.Equal(t, goPath+"/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.16.0", pkgPath)
}

func TestExternalPkgDirReplace(t *testing.T) {
	pkg := DefaultModule + "/model"
	pkgPath, _ := externalPackageDir(testDR(), pkg)
	assert.Equal(t, "../../../model", pkgPath)
}

func TestLocalPkg(t *testing.T) {
	pkg := pdata.NewSum()
	pkgValue := reflect.ValueOf(pkg)
	dr := testDR()
	output, _ := dr.PackageDir(pkgValue.Type())
	assert.Equal(t, "../../../model/pdata", output)
}
