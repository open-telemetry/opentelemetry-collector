// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRef(t *testing.T) {
	baseCfg := &Config{
		Namespace:   "github.com/open-telemetry/opentelemetry-collector-contrib",
		AllowedRefs: []string{"github.com/open-telemetry/opentelemetry-collector-contrib"},
	}

	tests := []struct {
		name      string
		path      string
		parentPkg string
		cfg       *Config
		expected  Ref
	}{
		{
			name:     "no dot — bare type name",
			path:     "MyType",
			cfg:      baseCfg,
			expected: Ref{typeName: "MyType"},
		},
		{
			name:     "absolute external package",
			path:     "github.com/some/pkg.SomeType",
			cfg:      baseCfg,
			expected: Ref{typeName: "SomeType", packageName: "github.com/some/pkg"},
		},
		{
			name:      "absolute-local path resolved via namespace",
			path:      "/receiver/fooreceiver.Config",
			parentPkg: "",
			cfg:       baseCfg,
			expected: Ref{
				typeName:    "Config",
				packageName: "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/fooreceiver",
				namespace:   "github.com/open-telemetry/opentelemetry-collector-contrib",
			},
		},
		{
			name:      "relative path with parentPkg",
			path:      "./subpkg.Config",
			parentPkg: "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/fooreceiver",
			cfg:       baseCfg,
			expected: Ref{
				typeName:    "Config",
				packageName: "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/fooreceiver/subpkg",
				namespace:   "github.com/open-telemetry/opentelemetry-collector-contrib",
			},
		},
		{
			name:      "relative path without parentPkg kept as-is",
			path:      "./subpkg.Config",
			parentPkg: "",
			cfg:       baseCfg,
			expected: Ref{
				typeName:    "Config",
				packageName: "./subpkg",
			},
		},
		{
			name: "allowed ref sets namespace",
			path: "github.com/open-telemetry/opentelemetry-collector-contrib/internal/shared.Type",
			cfg:  baseCfg,
			expected: Ref{
				typeName:    "Type",
				packageName: "github.com/open-telemetry/opentelemetry-collector-contrib/internal/shared",
				namespace:   "github.com/open-telemetry/opentelemetry-collector-contrib",
			},
		},
		{
			name: "package not in allowed refs — no namespace",
			path: "github.com/external/lib.Type",
			cfg:  baseCfg,
			expected: Ref{
				typeName:    "Type",
				packageName: "github.com/external/lib",
			},
		},
		{
			name: "multiple allowed refs — first prefix wins",
			path: "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/foo.Bar",
			cfg: &Config{
				Namespace: "github.com/open-telemetry/opentelemetry-collector-contrib",
				AllowedRefs: []string{
					"github.com/open-telemetry/opentelemetry-collector-contrib/pkg",
					"github.com/open-telemetry/opentelemetry-collector-contrib",
				},
			},
			expected: Ref{
				typeName:    "Bar",
				packageName: "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/foo",
				namespace:   "github.com/open-telemetry/opentelemetry-collector-contrib/pkg",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRef(tt.path, tt.parentPkg, tt.cfg)
			assert.Equal(t, tt.expected, *got)
		})
	}
}

// stubLoader is a schemaLoader that returns pre-built schemas by package name.
type stubLoader struct {
	schemas map[string]*Schema
}

func (s *stubLoader) Load(_ *Config, pattern string) (*Schema, error) {
	if sc, ok := s.schemas[pattern]; ok {
		return sc, nil
	}
	return nil, errors.New("stub: unknown package " + pattern)
}

func resolverWithStub(cfg *Config, schemas map[string]*Schema) *RefResolver {
	r := NewRefResolver(cfg)
	r.loader = &stubLoader{schemas: schemas}
	return r
}

func TestResolveLocalRef(t *testing.T) {
	cfg := &Config{Mode: Component, Namespace: "example.com/mod", AllowedRefs: []string{"example.com/mod"}}

	root := CreateSchema()
	inner := CreateObjectField("")
	inner.AddProperty("host", CreateSimpleField(SchemaTypeString, ""))
	root.Defs["inner_type"] = inner
	root.AddProperty("nested", CreateRefField("inner_type", ""))

	got, err := resolverWithStub(cfg, nil).Resolve(root)
	require.NoError(t, err)

	// Defs should be cleared in component mode after resolution
	assert.Nil(t, got.Defs)
	// The ref should have been replaced by the actual object
	resolved, ok := got.Properties["nested"].(*ObjectSchemaElement)
	require.True(t, ok)
	assert.Contains(t, resolved.Properties, "host")
}

func TestResolveExternalRef(t *testing.T) {
	cfg := &Config{Mode: Component, Namespace: "example.com/mod", AllowedRefs: []string{"example.com/mod"}}

	extSchema := CreateSchema()
	extObj := CreateObjectField("")
	extObj.AddProperty("port", CreateSimpleField(SchemaTypeInteger, ""))
	extSchema.Defs["ServerConfig"] = extObj

	root := CreateSchema()
	root.AddProperty("server", CreateRefField("example.com/mod/sub.ServerConfig", ""))

	got, err := resolverWithStub(cfg, map[string]*Schema{
		"example.com/mod/sub": extSchema,
	}).Resolve(root)
	require.NoError(t, err)

	resolved, ok := got.Properties["server"].(*ObjectSchemaElement)
	require.True(t, ok)
	assert.Contains(t, resolved.Properties, "port")
}

func TestResolvePreservesAnnotations(t *testing.T) {
	cfg := &Config{Mode: Component, Namespace: "example.com/mod", AllowedRefs: []string{"example.com/mod"}}

	root := CreateSchema()
	inner := CreateObjectField("")
	inner.AddProperty("x", CreateSimpleField(SchemaTypeString, ""))
	root.Defs["inner_type"] = inner

	ref := CreateRefField("inner_type", "original description")
	ref.IsOptional = true
	ref.IsPointer = true
	root.AddProperty("field", ref)

	got, err := resolverWithStub(cfg, nil).Resolve(root)
	require.NoError(t, err)

	resolved := got.Properties["field"]
	assert.Equal(t, "original description", resolved.(*ObjectSchemaElement).Description)
	assert.True(t, resolved.(*ObjectSchemaElement).IsOptional)
	assert.True(t, resolved.(*ObjectSchemaElement).IsPointer)
}

func TestNotFoundError(t *testing.T) {
	assert.Equal(t, `reference type "Foo" not found`, (&NotFoundError{TypeName: "Foo"}).Error())
	assert.Equal(t, `reference type "Foo" not found in package "a/b"`, (&NotFoundError{TypeName: "Foo", PackageName: "a/b"}).Error())
}

func TestResolveExternalRefNotFoundDropsProperty(t *testing.T) {
	cfg := &Config{Mode: Component, Namespace: "example.com/mod", AllowedRefs: []string{"example.com/mod"}}

	extSchema := CreateSchema()
	extObj := CreateObjectField("")
	extObj.AddProperty("port", CreateSimpleField(SchemaTypeInteger, ""))
	extSchema.Defs["GoodType"] = extObj

	root := CreateSchema()
	root.AddProperty("good", CreateRefField("example.com/mod/sub.GoodType", ""))
	root.AddProperty("bad", CreateRefField("example.com/mod/sub.Missing", ""))

	got, err := resolverWithStub(cfg, map[string]*Schema{
		"example.com/mod/sub": extSchema,
	}).Resolve(root)
	require.NoError(t, err)
	assert.Contains(t, got.Properties, "good")
	assert.NotContains(t, got.Properties, "bad")
}

func TestResolveNotFoundDropsAllOfEntry(t *testing.T) {
	cfg := &Config{Mode: Component, Namespace: "example.com/mod", AllowedRefs: []string{"example.com/mod"}}

	extSchema := CreateSchema()
	extSchema.Defs["Good"] = CreateObjectField("")

	root := CreateSchema()
	good := CreateRefField("example.com/mod/sub.Good", "")
	bad := CreateRefField("example.com/mod/sub.Missing", "")
	root.AllOf = []SchemaElement{good, bad}

	got, err := resolverWithStub(cfg, map[string]*Schema{
		"example.com/mod/sub": extSchema,
	}).Resolve(root)
	require.NoError(t, err)
	assert.Len(t, got.AllOf, 1)
}

func TestResolveNotFoundDropsAdditionalProperties(t *testing.T) {
	cfg := &Config{Mode: Component, Namespace: "example.com/mod", AllowedRefs: []string{"example.com/mod"}}

	extSchema := CreateSchema() // no Defs

	root := CreateSchema()
	root.AdditionalProperties = CreateRefField("example.com/mod/sub.Missing", "")

	got, err := resolverWithStub(cfg, map[string]*Schema{
		"example.com/mod/sub": extSchema,
	}).Resolve(root)
	require.NoError(t, err)
	assert.Nil(t, got.AdditionalProperties)
}

func TestResolveNotFoundDropsArrayProperty(t *testing.T) {
	cfg := &Config{Mode: Component, Namespace: "example.com/mod", AllowedRefs: []string{"example.com/mod"}}

	extSchema := CreateSchema() // no Defs

	root := CreateSchema()
	root.AddProperty("items", CreateArrayField(CreateRefField("example.com/mod/sub.Missing", ""), ""))

	got, err := resolverWithStub(cfg, map[string]*Schema{
		"example.com/mod/sub": extSchema,
	}).Resolve(root)
	require.NoError(t, err)
	assert.NotContains(t, got.Properties, "items")
}

func TestResolveLoaderError(t *testing.T) {
	cfg := &Config{Mode: Component, Namespace: "example.com/mod", AllowedRefs: []string{"example.com/mod"}}

	root := CreateSchema()
	root.AddProperty("x", CreateRefField("example.com/mod/missing.Type", ""))

	_, err := resolverWithStub(cfg, map[string]*Schema{}).Resolve(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load ref schema")
	assert.Contains(t, err.Error(), "example.com/mod/missing")
}

func TestResolveLocalRefNotFoundDropsProperty(t *testing.T) {
	cfg := &Config{Mode: Component, Namespace: "example.com/mod", AllowedRefs: []string{"example.com/mod"}}

	root := CreateSchema()
	// "unknown_type" is not registered in Defs
	root.AddProperty("bad", CreateRefField("unknown_type", ""))

	got, err := resolverWithStub(cfg, nil).Resolve(root)
	require.NoError(t, err)
	assert.NotContains(t, got.Properties, "bad")
}

func TestResolveArrayRef(t *testing.T) {
	cfg := &Config{Mode: Component, Namespace: "example.com/mod", AllowedRefs: []string{"example.com/mod"}}

	root := CreateSchema()
	inner := CreateObjectField("")
	inner.AddProperty("name", CreateSimpleField(SchemaTypeString, ""))
	root.Defs["item"] = inner
	root.AddProperty("items", CreateArrayField(CreateRefField("item", ""), ""))

	got, err := resolverWithStub(cfg, nil).Resolve(root)
	require.NoError(t, err)

	arr, ok := got.Properties["items"].(*ArraySchemaElement)
	require.True(t, ok)
	itemObj, ok := arr.Items.(*ObjectSchemaElement)
	require.True(t, ok)
	assert.Contains(t, itemObj.Properties, "name")
}
