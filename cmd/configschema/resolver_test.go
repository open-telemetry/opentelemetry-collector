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
	output, err := dr.PackageDir(pkgValue.Type())
	assert.NoError(t, err)
	assert.Equal(t, "../../model/pdata", output)
}

func TestPackageDirError(t *testing.T) {
	pkg := pdata.NewSum()
	pkgType := reflect.ValueOf(pkg).Type()
	srcRoot := "test/fail"
	dr := NewDirResolver(srcRoot, DefaultModule)
	output, err := dr.PackageDir(pkgType)
	assert.Error(t, err)
	assert.Equal(t, "", output)
}

func TestExternalPkgDirErr(t *testing.T) {
	pkg := "random/test"
	pkgPath, err := testDR().externalPackageDir(pkg)
	if assert.Error(t, err) {
		expected := fmt.Sprintf("could not find package: \"%s\"", pkg)
		assert.EqualErrorf(t, err, expected, "")
	}
	assert.Equal(t, pkgPath, "")
}

func TestExternalPkgDir(t *testing.T) {
	dr := testDR()
	testPkg := "grpc-ecosystem/grpc-gateway"
	pkgPath, err := dr.externalPackageDir(testPkg)
	assert.NoError(t, err)
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		goPath = build.Default.GOPATH
	}
	testLine, testVers, err := grepMod(path.Join(dr.SrcRoot, "go.mod"), testPkg)
	assert.NoError(t, err)
	expected := fmt.Sprint(path.Join(goPath, "pkg", "mod", testLine+"@"+testVers))
	assert.Equal(t, expected, pkgPath)
}

func TestExternalPkgDirReplace(t *testing.T) {
	pkg := DefaultModule + "/model"
	pkgPath, err := testDR().externalPackageDir(pkg)
	assert.NoError(t, err)
	assert.Equal(t, "../../model", pkgPath)
}
