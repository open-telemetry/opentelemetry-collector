//go:build !windows

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package internal

import (
	"container/list"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponentParser(t *testing.T) {
	type testCase struct {
		title              string
		inputFile          string
		expectedSchemaFile string
		rootType           string
	}

	testCases := []testCase{
		{
			title:              "Test Simple Config Parsing",
			inputFile:          "testdata/test00/SimpleConfig.go",
			expectedSchemaFile: "testdata/test00/simple_config.schema.yaml",
			rootType:           "SimpleConfig",
		},
		{
			title:              "Test Array field Config Parsing",
			inputFile:          "testdata/test01/ArrayFieldConfig.go",
			expectedSchemaFile: "testdata/test01/array_field_config.schema.yaml",
			rootType:           "SimpleArrayConfig",
		},
		{
			title:              "Test Nested Struct Config Parsing",
			inputFile:          "testdata/test02/NestedStructConfig.go",
			expectedSchemaFile: "testdata/test02/nested_struct_config.schema.yaml",
			rootType:           "Config",
		},
		{
			title:              "Test Map field Config Parsing",
			inputFile:          "testdata/test03/MapFieldConfig.go",
			expectedSchemaFile: "testdata/test03/map_field_config.schema.yaml",
			rootType:           "MapConfig",
		},
		{
			title:              "Test Ref field Config Parsing",
			inputFile:          "testdata/test04/RefFieldConfig.go",
			expectedSchemaFile: "testdata/test04/ref_field_config.schema.yaml",
			rootType:           "RefFieldConfig",
		},
		{
			title:              "Test Embedded Struct Config Parsing",
			inputFile:          "testdata/test05/EmbeddedStructConfig.go",
			expectedSchemaFile: "testdata/test05/embedded_struct_config.schema.yaml",
			rootType:           "EmbeddedStructConfig",
		},
		{
			title:              "Test Pointer field Config Parsing",
			inputFile:          "testdata/test06/PointerFieldConfig.go",
			expectedSchemaFile: "testdata/test06/pointer_field_config.schema.yaml",
			rootType:           "PointerFieldConfig",
		},
		{
			title:              "Test complex type field Config Parsing",
			inputFile:          "testdata/test07/ComplexTypeFieldConfig.go",
			expectedSchemaFile: "testdata/test07/complex_type_field_config.schema.yaml",
			rootType:           "ComplexTypeFieldConfig",
		},
		{
			title:              "Test time type fields Config Parsing",
			inputFile:          "testdata/test08/TimeTypeFieldConfig.go",
			expectedSchemaFile: "testdata/test08/time_type_field_config.schema.yaml",
			rootType:           "TimeTypeFieldConfig",
		},
		{
			title:              "Test Mixed Tags Config Parsing",
			inputFile:          "testdata/test09/MixedTagsConfig.go",
			expectedSchemaFile: "testdata/test09/mixed_tags_config.schema.yaml",
			rootType:           "MixedTagsConfig",
		},
		{
			title:              "Test Simple Type Aliases Parsing",
			inputFile:          "testdata/test10/AliasSimpleTypeConfig.go",
			expectedSchemaFile: "testdata/test10/alias_simple_type_config.schema.yaml",
			rootType:           "AliasSimpleTypeConfig",
		},
		{
			title:              "Test External Refs Parsing",
			inputFile:          "testdata/test11/ExternalRefsConfig.go",
			expectedSchemaFile: "testdata/test11/external_refs_config.schema.yaml",
			rootType:           "ExternalRefsConfig",
		},
		{
			title:              "Test interface fields Parsing",
			inputFile:          "testdata/test12/InterfaceFieldConfig.go",
			expectedSchemaFile: "testdata/test12/interface_field_config.schema.yaml",
			rootType:           "InterfaceFieldConfig",
		},
		{
			title:              "Test named map types Parsing",
			inputFile:          "testdata/test13/NamedMapConfig.go",
			expectedSchemaFile: "testdata/test13/named_map_config.schema.yaml",
			rootType:           "NamedMapConfig",
		},
		{
			title:              "Test remain field with map type Parsing",
			inputFile:          "testdata/test14/RemainMapConfig.go",
			expectedSchemaFile: "testdata/test14/remain_map_config.schema.yaml",
			rootType:           "RemainMapConfig",
		},
		{
			title:              "Test remain field with non-map type Parsing",
			inputFile:          "testdata/test15/RemainAnyConfig.go",
			expectedSchemaFile: "testdata/test15/remain_any_config.schema.yaml",
			rootType:           "RemainAnyConfig",
		},
		{
			title:              "Test UnmarshalText types Parsing",
			inputFile:          "testdata/test16/TextUnmarshalerConfig.go",
			expectedSchemaFile: "testdata/test16/text_unmarshaler_config.schema.yaml",
			rootType:           "TextUnmarshalerConfig",
		},
		{
			title:              "Test named interface, slice and map types Parsing",
			inputFile:          "testdata/test17/NamedTypesConfig.go",
			expectedSchemaFile: "testdata/test17/named_types_config.schema.yaml",
			rootType:           "NamedTypesConfig",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			expectedBytes, err := os.ReadFile(tc.expectedSchemaFile)
			if err != nil {
				t.Fatalf("Failed to read expected schema file %s: %v", tc.expectedSchemaFile, err)
			}
			expectedSchema := string(expectedBytes)

			dir, _ := filepath.Abs(filepath.Dir(tc.inputFile))
			cfg := &Config{
				Mode:     Component,
				DirPath:  dir,
				Mappings: testMappings(),
				AllowedRefs: []string{
					"go.opentelemetry.io/collector",
				},
				Namespace: "go.opentelemetry.io/collector",
				Pattern:   ".",
			}
			if tc.rootType != "" {
				cfg.ConfigType = tc.rootType
			}
			parser := NewParser(cfg)

			schema, err := parser.Parse()
			require.NoError(t, err)

			rawYaml, err := schema.ToYAML()
			require.NoError(t, err)

			givenYaml := string(rawYaml)
			require.YAMLEq(t, expectedSchema, givenYaml)
		})
	}
}

func TestPackageParser(t *testing.T) {
	dir, _ := filepath.Abs("testdata/external/")
	cfg := &Config{
		Mode:     Package,
		DirPath:  dir,
		Mappings: testMappings(),
		Pattern:  ".",
	}

	parser := NewParser(cfg)

	schema, err := parser.Parse()
	require.NoError(t, err)

	rawYaml, err := schema.ToYAML()
	require.NoError(t, err)

	expectedBytes, err := os.ReadFile("testdata/external/config.schema.yaml")
	if err != nil {
		t.Fatalf("Failed to read expected schema file: %v", err)
	}
	expectedSchema := string(expectedBytes)

	givenYaml := string(rawYaml)
	require.YAMLEq(t, expectedSchema, givenYaml)
}

func TestParsePatternNoExportedTypes(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/noexport\n\ngo 1.20\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.go"), []byte(`package noexport

type config struct{}
`), 0o600))

	parser := NewParser(&Config{
		Mode:    Package,
		DirPath: dir,
	})

	schema, err := parser.ParsePattern(".")
	require.Nil(t, schema)
	require.EqualError(t, err, "no exported types found in the package")
}

func testMappings() Mappings {
	return Mappings{
		"time": PackagesMapping{
			"Time": TypeDesc{
				SchemaType: SchemaTypeString,
				Format:     "date-time",
			},
			"Duration": TypeDesc{
				SchemaType: SchemaTypeString,
				Format:     "duration",
			},
		},
	}
}

func TestFeedProcessQueueErrorMessage(t *testing.T) {
	tests := []struct {
		name             string
		pattern          string
		wantPatternInMsg bool
	}{
		{
			name:             "dot pattern omits pattern from error",
			pattern:          ".",
			wantPatternInMsg: false,
		},
		{
			name:             "explicit pattern included in error",
			pattern:          "example.com/mod/foo",
			wantPatternInMsg: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				config: &Config{
					Mode:     Component,
					Mappings: testMappings(),
				},
				types:        map[string]*TypeInfo{},
				processQueue: list.New(),
			}
			err := p.feedProcessQueue(tt.pattern)
			require.Error(t, err)
			if tt.wantPatternInMsg {
				assert.Contains(t, err.Error(), tt.pattern)
			} else {
				assert.NotContains(t, err.Error(), tt.pattern)
			}
		})
	}
}

func TestParseTypeNilGuard(t *testing.T) {
	src := `package test

// MyConfig is a config type backed by an unrecognized generic.
type MyConfig pkg.Unknown[string]
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	require.NoError(t, err)

	cmap := ast.NewCommentMap(fset, file, file.Comments)
	var typeInfo *TypeInfo
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if ts.Name.Name == "MyConfig" {
				typeInfo = &TypeInfo{
					spec:     ts,
					comms:    cmap[genDecl],
					typeName: ts.Name.Name,
					imports:  map[string]string{},
				}
			}
		}
	}
	require.NotNil(t, typeInfo, "TypeInfo for MyConfig not found")
	require.NotEmpty(t, typeInfo.comms, "expected doc comment to be present")

	p := &Parser{
		config: &Config{
			Mode:     Component,
			Mappings: testMappings(),
		},
		types: map[string]*TypeInfo{},
	}

	elem, err := p.parseType(typeInfo)
	require.NoError(t, err)
	require.Nil(t, elem)
}

func TestTextUnmarshalerReceiver(t *testing.T) {
	src := `package test

func (l *Level) UnmarshalText(_ []byte) error { return nil }

func (m Mode) UnmarshalText(_ []byte) error { return nil }

func (l Level) String() string { return "" }

func UnmarshalText(_ []byte) error { return nil }

func (g *Generic[T]) UnmarshalText(_ []byte) error { return nil }
`
	file, err := parser.ParseFile(token.NewFileSet(), "test.go", src, 0)
	require.NoError(t, err)

	var got []string
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		require.True(t, ok)
		if name, isText := textUnmarshalerReceiver(funcDecl); isText {
			got = append(got, name)
		}
	}
	// A method with another name, a function with no receiver and a method
	// on a generic type are not reported.
	assert.Equal(t, []string{"Level", "Mode"}, got)
}

func TestParseTypeUnsupportedNamedType(t *testing.T) {
	src := `package test

type Events chan string
`
	file, err := parser.ParseFile(token.NewFileSet(), "test.go", src, 0)
	require.NoError(t, err)
	typeSpec := file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)

	p := NewParser(&Config{})
	_, err = p.parseType(&TypeInfo{spec: typeSpec, typeName: "Events", imports: map[string]string{}})
	require.ErrorContains(t, err, "unrecognized field type")
}

// The resolver walks the definitions in map order. The output must not depend
// on that order and must not keep any reference.
func TestResolveNamedTypesIsStable(t *testing.T) {
	dir, err := filepath.Abs("testdata/test17")
	require.NoError(t, err)

	for _, mode := range []RunMode{Component, Package} {
		t.Run(string(mode), func(t *testing.T) {
			var first string
			for range 20 {
				cfg := &Config{
					Mode:        mode,
					DirPath:     dir,
					Mappings:    testMappings(),
					Namespace:   "go.opentelemetry.io/collector",
					Pattern:     ".",
					ConfigType:  "NamedTypesConfig",
					ResolveRefs: true,
				}
				schema, err := NewParser(cfg).Parse()
				require.NoError(t, err)
				schema, err = NewRefResolver(cfg).Resolve(schema)
				require.NoError(t, err)

				_, err = schema.ToJSON()
				require.NoError(t, err)
				raw, err := schema.ToYAML()
				require.NoError(t, err)
				got := string(raw)
				require.NotContains(t, got, "$ref")
				if first == "" {
					first = got
				}
				require.Equal(t, first, got)
			}
			if mode == Component {
				expected, err := os.ReadFile("testdata/test17/named_types_config.resolved.schema.yaml")
				require.NoError(t, err)
				require.YAMLEq(t, string(expected), first)
			}
		})
	}
}
