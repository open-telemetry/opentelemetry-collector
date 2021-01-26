// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filterexpr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestToIlExpr(t *testing.T) {
	expr, err := CompileIlExpr(`Name != "test" || Name == "test" && Version ~= "^ver.*"`)
	require.NoError(t, err)
	assert.True(t, expr.Evaluate(newIl("name", "")))
	assert.True(t, expr.Evaluate(newIl("test", "version")))
	assert.False(t, expr.Evaluate(newIl("test", "subver")))
}

func TestToIlExpr_Complex(t *testing.T) {
	expr, err := CompileIlExpr(
		`(Name != "name" && Name != "other") || !( Name != "name" || Version != "version" ) || (Name == "other" && Version == "testver" )`)
	require.NoError(t, err)
	assert.False(t, expr.Evaluate(newIl("name", "")))
	assert.True(t, expr.Evaluate(newIl("name", "version")))
	assert.False(t, expr.Evaluate(newIl("other", "")))
	assert.True(t, expr.Evaluate(newIl("other", "testver")))
}

func TestToIlExpr_NotRegexp(t *testing.T) {
	expr, err := CompileIlExpr(`!( Name ~= "^te.*" )`)
	require.NoError(t, err)
	assert.True(t, expr.Evaluate(newIl("name", "")))
	assert.False(t, expr.Evaluate(newIl("test", "")))
}

func TestToIlExpr_Empty(t *testing.T) {
	expr, err := CompileIlExpr(``)
	require.NoError(t, err)
	assert.Nil(t, expr)
}

func TestCompileIlError(t *testing.T) {
	testcases := []struct {
		name       string
		exprString string
	}{
		{
			name:       "And_NameRegexp",
			exprString: `Name ~= "[a-z)*" && Version ~= "[a-z]*"`,
		},
		{
			name:       "And_VersionRegexp",
			exprString: `Name ~= "[a-z]*" && Version ~= "[a-z)*"`,
		},
		{
			name:       "Or_NameRegexp",
			exprString: `Name ~= "[a-z)*" || Version ~= "[a-z]*"`,
		},
		{
			name:       "Or_VersionRegexp",
			exprString: `Name ~= "[a-z]*" || Version ~= "[a-z)*"`,
		},
		{
			name:       "Not_NameRegexp",
			exprString: `!(Name ~= "[a-z)*")`,
		},
		{
			name:       "invalid_name",
			exprString: `Naame != "test"`,
		},
		{
			name:       "invalid_version",
			exprString: `Veersion == "test"`,
		},
		{
			name:       "invalid_id",
			exprString: `Id ~= "test"`,
		},
		{
			name:       "invalid_order",
			exprString: `"test" == Name`,
		},
		{
			name:       "invalid_not",
			exprString: `!Name == "test"`,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CompileIlExpr(tc.exprString)
			assert.Error(t, err)
		})
	}
}

func newIl(name, version string) pdata.InstrumentationLibrary {
	ils := pdata.NewInstrumentationLibrary()
	ils.SetName(name)
	ils.SetVersion(version)
	return ils
}

func BenchmarkEvaluateIlExpr(b *testing.B) {
	library := pdata.NewInstrumentationLibrary()
	library.SetName("lib")
	library.SetVersion("ver")

	ils, err := CompileIlExpr(`Name == "lib" && Version == "ver"`)
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if !ils.Evaluate(library) {
			b.Fail()
		}
	}
}
