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
	"fmt"
	"go/build"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/model/pdata"
)

func TestPackageDirLocal(t *testing.T) {
	pkg := pdata.NewSum()
	pkgValue := reflect.ValueOf(pkg)
	dr := testDR()
	output, _ := dr.PackageDir(pkgValue.Type())
	assert.Equal(t, "../../model/pdata", output)
}

func TestPackageDirError(t *testing.T) {
	pkg := pdata.NewSum()
	pkgType := reflect.ValueOf(pkg).Type()
	dr := NewDirResolver("test/fail", DefaultModule)
	output, err := dr.PackageDir(pkgType)
	assert.NotEqual(t, nil, err)
	assert.Equal(t, "", output)
}

func TestExternalPkgDirErr(t *testing.T) {
	pkgPath, err := testDR().externalPackageDir("random/test")
	assert.NotEqual(t, err, nil)
	assert.Equal(t, pkgPath, "")
}

func TestExternalPkgDir(t *testing.T) {
	dr := testDR()
	testPkg := "grpc-ecosystem/grpc-gateway"
	pkgPath, _ := dr.externalPackageDir(testPkg)
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		goPath = build.Default.GOPATH
	}
	testLine, testVers, _ := grepMod(path.Join(dr.SrcRoot, "go.mod"), testPkg)
	expected := fmt.Sprint(path.Join(goPath, "pkg", "mod", testLine+"@"+testVers))
	assert.Equal(t, expected, pkgPath)
}

func TestExternalPkgDirReplace(t *testing.T) {
	pkg := DefaultModule + "/model"
	pkgPath, _ := testDR().externalPackageDir(pkg)
	assert.Equal(t, "../../model", pkgPath)
}
