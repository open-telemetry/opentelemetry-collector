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
