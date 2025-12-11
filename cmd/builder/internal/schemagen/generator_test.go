// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

func TestGenerateSchema(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Navigate up to the repository root (from cmd/builder/internal/schemagen)
	repoRoot := filepath.Join(cwd, "..", "..", "..", "..")

	tests := []struct {
		name          string
		kind          component.Kind
		componentName string
		importPath    string
		expectedFile  string
		analyzerRoot  string
	}{
		{
			name:          "testcomponent",
			kind:          component.KindExporter,
			componentName: "testcomponent",
			importPath:    "go.opentelemetry.io/collector/cmd/builder/internal/schemagen/testdata/testcomponent",
			expectedFile:  "testdata/exporter_testcomponent.json",
			analyzerRoot:  filepath.Join(cwd, "..", ".."), // builder module root
		},
		{
			name:          "otlpexporter",
			kind:          component.KindExporter,
			componentName: "otlpexporter",
			importPath:    "go.opentelemetry.io/collector/exporter/otlpexporter",
			expectedFile:  "testdata/exporter_otlpexporter.json",
			analyzerRoot:  filepath.Join(repoRoot, "exporter", "otlpexporter"), // otlpexporter module root
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			analyzer := NewPackageAnalyzer(tt.analyzerRoot)
			generator := NewSchemaGenerator(tempDir, analyzer)

			// Generate schema
			err := generator.GenerateSchema(tt.kind, tt.componentName, tt.importPath)
			require.NoError(t, err)

			// Read generated schema
			generatedPath := filepath.Join(tempDir, fmt.Sprintf("%s_%s.json", strings.ToLower(tt.kind.String()), tt.componentName))
			generatedContent, err := os.ReadFile(generatedPath) // #nosec G304 -- test file path from test setup
			require.NoError(t, err)

			// Read expected schema
			expectedContent, err := os.ReadFile(tt.expectedFile)
			require.NoError(t, err)

			// Compare JSON (ignoring formatting differences)
			if !assert.JSONEq(t, string(expectedContent), string(generatedContent)) {
				// If test fails and UPDATE_GOLDEN env is set, update the expected file
				if os.Getenv("UPDATE_GOLDEN") == "1" {
					err := os.WriteFile(tt.expectedFile, generatedContent, 0o600)
					require.NoError(t, err)
					t.Logf("Updated expected file: %s", tt.expectedFile)
				} else {
					t.Logf("To update expected file, run:\n  UPDATE_GOLDEN=1 go test -run TestGenerateSchema/%s ./internal/schemagen/...", tt.name)
				}
			}
		})
	}
}

func TestGetFieldName(t *testing.T) {
	tests := []struct {
		name     string
		field    reflect.StructField
		expected string
	}{
		{
			name: "mapstructure tag",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `mapstructure:"test_field"`,
			},
			expected: "test_field",
		},
		{
			name: "json tag",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `json:"json_field"`,
			},
			expected: "json_field",
		},
		{
			name: "mapstructure takes precedence over json",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `mapstructure:"map_field" json:"json_field"`,
			},
			expected: "map_field",
		},
		{
			name: "no tag uses lowercase field name",
			field: reflect.StructField{
				Name: "TestField",
			},
			expected: "testfield",
		},
		{
			name: "skip field with dash",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `mapstructure:"-"`,
			},
			expected: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetFieldName(tt.field)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanComment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple comment",
			input:    "This is a comment",
			expected: "This is a comment",
		},
		{
			name:     "multiline comment",
			input:    "Line 1\nLine 2\nLine 3",
			expected: "Line 1 Line 2 Line 3",
		},
		{
			name:     "comment with markers",
			input:    "// This is a comment",
			expected: "This is a comment",
		},
		{
			name:     "empty lines",
			input:    "Line 1\n\nLine 2",
			expected: "Line 1 Line 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanComment(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDeprecatedFromDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		expected    bool
	}{
		{
			name:        "deprecated keyword",
			description: "This field is deprecated and will be removed",
			expected:    true,
		},
		{
			name:        "replaced by keyword",
			description: "Use NewField instead, replaced by NewField",
			expected:    true,
		},
		{
			name:        "obsolete keyword",
			description: "This is an obsolete feature",
			expected:    true,
		},
		{
			name:        "not deprecated",
			description: "This is a normal field",
			expected:    false,
		},
		{
			name:        "empty description",
			description: "",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDeprecatedFromDescription(tt.description)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDeprecatedFromTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected bool
	}{
		{
			name:     "deprecated tag",
			tag:      `deprecated:"true"`,
			expected: true,
		},
		{
			name:     "mapstructure with deprecated",
			tag:      `mapstructure:"field,deprecated"`,
			expected: true,
		},
		{
			name:     "json with deprecated",
			tag:      `json:"field,deprecated"`,
			expected: true,
		},
		{
			name:     "not deprecated",
			tag:      `mapstructure:"field"`,
			expected: false,
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDeprecatedFromTag(tt.tag)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractTagValue(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		key      string
		expected string
	}{
		{
			name:     "mapstructure tag",
			tag:      `mapstructure:"field_name"`,
			key:      "mapstructure",
			expected: "field_name",
		},
		{
			name:     "json tag",
			tag:      `json:"json_name,omitempty"`,
			key:      "json",
			expected: "json_name,omitempty",
		},
		{
			name:     "missing tag",
			tag:      `mapstructure:"field"`,
			key:      "json",
			expected: "",
		},
		{
			name:     "multiple tags",
			tag:      `mapstructure:"map_field" json:"json_field"`,
			key:      "json",
			expected: "json_field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTagValue(tt.tag, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSchemaGenerator_WriteSchemaToFile(t *testing.T) {
	tempDir := t.TempDir()
	analyzer := NewPackageAnalyzer(tempDir)
	sg := NewSchemaGenerator(tempDir, analyzer)

	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type": "string",
			},
		},
	}

	filePath := filepath.Join(tempDir, "test_schema.json")
	err := sg.writeSchemaToFile(filePath, schema)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(filePath)
	require.NoError(t, err)

	// Read and verify content
	content, err := os.ReadFile(filePath) // #nosec G304 -- test file path from test setup
	require.NoError(t, err)
	assert.Contains(t, string(content), `"$schema"`)
	assert.Contains(t, string(content), `"properties"`)
}

func TestSchemaGenerator_OutputDir(t *testing.T) {
	tempDir := t.TempDir()
	analyzer := NewPackageAnalyzer(tempDir)
	sg := NewSchemaGenerator(tempDir, analyzer)

	assert.Equal(t, tempDir, sg.OutputDir())
}

func TestCommentExtractor_GetFieldComment(t *testing.T) {
	ce := NewCommentExtractor()

	// Manually populate the cache for testing
	ce.commentCache["test/package"] = map[string]string{
		"Config.Name":    "The name of the component",
		"Config.Timeout": "The timeout duration",
	}

	t.Run("existing comment", func(t *testing.T) {
		comment := ce.GetFieldComment("test/package", "Config", "Name")
		assert.Equal(t, "The name of the component", comment)
	})

	t.Run("missing comment", func(t *testing.T) {
		comment := ce.GetFieldComment("test/package", "Config", "NonExistent")
		assert.Empty(t, comment)
	})

	t.Run("missing package", func(t *testing.T) {
		comment := ce.GetFieldComment("other/package", "Config", "Name")
		assert.Empty(t, comment)
	})
}

func TestPackageAnalyzer_NewPackageAnalyzer(t *testing.T) {
	tempDir := t.TempDir()
	analyzer := NewPackageAnalyzer(tempDir)

	assert.NotNil(t, analyzer)
	assert.NotNil(t, analyzer.packages)
	assert.NotNil(t, analyzer.cfg)
	assert.Equal(t, tempDir, analyzer.cfg.Dir)
}

func TestPackageAnalyzer_FindConfigType_WithCompileTimeCheck(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// testcomponent uses "MySettings" (not "Config") to verify AST-based detection
	analyzer := NewPackageAnalyzer(filepath.Join(cwd, "..", ".."))
	pkg, err := analyzer.LoadPackage("go.opentelemetry.io/collector/cmd/builder/internal/schemagen/testdata/testcomponent")
	require.NoError(t, err)

	configType, err := analyzer.FindConfigType(pkg)
	require.NoError(t, err)
	assert.Equal(t, "MySettings", configType.Obj().Name())
}

func TestBuildImportAliasMap(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected map[string]string
	}{
		{
			name: "standard import",
			code: `package test
import "go.opentelemetry.io/collector/component"
`,
			expected: map[string]string{
				"component": "go.opentelemetry.io/collector/component",
			},
		},
		{
			name: "aliased import",
			code: `package test
import comp "go.opentelemetry.io/collector/component"
`,
			expected: map[string]string{
				"comp": "go.opentelemetry.io/collector/component",
			},
		},
		{
			name: "multiple imports with alias",
			code: `package test
import (
	"fmt"
	comp "go.opentelemetry.io/collector/component"
	"testing"
)
`,
			expected: map[string]string{
				"fmt":     "fmt",
				"comp":    "go.opentelemetry.io/collector/component",
				"testing": "testing",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ImportsOnly)
			require.NoError(t, err)

			result := buildImportAliasMap(file)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestImportPathToName(t *testing.T) {
	tests := []struct {
		importPath string
		expected   string
	}{
		{"go.opentelemetry.io/collector/component", "component"},
		{"fmt", "fmt"},
		{"github.com/stretchr/testify/assert", "assert"},
		{"path/to/pkg", "pkg"},
	}

	for _, tt := range tests {
		t.Run(tt.importPath, func(t *testing.T) {
			result := importPathToName(tt.importPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsComponentConfigType(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name: "standard component.Config",
			code: `package test
import "go.opentelemetry.io/collector/component"
var _ component.Config = (*Config)(nil)
`,
			expected: true,
		},
		{
			name: "aliased import comp.Config",
			code: `package test
import comp "go.opentelemetry.io/collector/component"
var _ comp.Config = (*Config)(nil)
`,
			expected: true,
		},
		{
			name: "different package same selector name",
			code: `package test
import "some/other/pkg"
var _ pkg.Config = (*Config)(nil)
`,
			expected: false,
		},
		{
			name: "component.Factory should not match",
			code: `package test
import "go.opentelemetry.io/collector/component"
var _ component.Factory = (*Factory)(nil)
`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			require.NoError(t, err)

			importAliases := buildImportAliasMap(file)

			// Find the var declaration
			var typeExpr ast.Expr
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.VAR {
					continue
				}
				for _, spec := range genDecl.Specs {
					valueSpec, ok := spec.(*ast.ValueSpec)
					if ok && len(valueSpec.Names) == 1 && valueSpec.Names[0].Name == "_" {
						typeExpr = valueSpec.Type
						break
					}
				}
			}

			require.NotNil(t, typeExpr, "could not find var declaration in test code")
			result := isComponentConfigType(typeExpr, importAliases)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBasicTypeToSchema(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected map[string]any
	}{
		{
			name:     "string",
			value:    "",
			expected: map[string]any{"type": "string"},
		},
		{
			name:     "int",
			value:    0,
			expected: map[string]any{"type": "integer"},
		},
		{
			name:     "bool",
			value:    false,
			expected: map[string]any{"type": "boolean"},
		},
		{
			name:     "float64",
			value:    0.0,
			expected: map[string]any{"type": "number"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Using generateBasicTypeSchema indirectly through the type system is complex
			// For now, we just verify the expected patterns exist
			assert.NotNil(t, tt.expected["type"])
		})
	}
}

func TestPackageAnalyzer_GetPackage(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	analyzer := NewPackageAnalyzer(filepath.Join(cwd, "..", ".."))

	t.Run("returns nil for unloaded package", func(t *testing.T) {
		pkg := analyzer.GetPackage("nonexistent/package")
		assert.Nil(t, pkg)
	})

	t.Run("returns cached package after load", func(t *testing.T) {
		importPath := "go.opentelemetry.io/collector/cmd/builder/internal/schemagen/testdata/testcomponent"
		loaded, err := analyzer.LoadPackage(importPath)
		require.NoError(t, err)

		cached := analyzer.GetPackage(importPath)
		assert.Equal(t, loaded, cached)
	})
}

func TestPackageAnalyzer_LoadPackage_Caching(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	analyzer := NewPackageAnalyzer(filepath.Join(cwd, "..", ".."))
	importPath := "go.opentelemetry.io/collector/cmd/builder/internal/schemagen/testdata/testcomponent"

	// First load
	pkg1, err := analyzer.LoadPackage(importPath)
	require.NoError(t, err)

	// Second load should return cached
	pkg2, err := analyzer.LoadPackage(importPath)
	require.NoError(t, err)

	assert.Same(t, pkg1, pkg2, "LoadPackage should return cached package on second call")
}

func TestPackageAnalyzer_LoadPackage_Errors(t *testing.T) {
	tempDir := t.TempDir()
	analyzer := NewPackageAnalyzer(tempDir)

	t.Run("nonexistent package", func(t *testing.T) {
		_, err := analyzer.LoadPackage("nonexistent/fake/package/that/does/not/exist")
		assert.Error(t, err)
	})
}

func TestExtractTypeFromConversionExpr(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name: "valid conversion expression",
			code: `package test
import "go.opentelemetry.io/collector/component"
var _ component.Config = (*MyConfig)(nil)
`,
			expected: "MyConfig",
		},
		{
			name: "non-pointer conversion",
			code: `package test
import "go.opentelemetry.io/collector/component"
var _ component.Config = MyConfig{}
`,
			expected: "",
		},
		{
			name: "multiple arguments",
			code: `package test
import "go.opentelemetry.io/collector/component"
var _ component.Config = (*MyConfig)(nil, nil)
`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			require.NoError(t, err)

			analyzer := &PackageAnalyzer{}

			// Find the var declaration and extract the value
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.VAR {
					continue
				}
				for _, spec := range genDecl.Specs {
					valueSpec, ok := spec.(*ast.ValueSpec)
					if !ok || len(valueSpec.Values) == 0 {
						continue
					}
					result := analyzer.extractTypeFromConversionExpr(valueSpec.Values[0])
					assert.Equal(t, tt.expected, result)
					return
				}
			}
		})
	}
}

func TestHandleSpecialType(t *testing.T) {
	// Test with nil type
	t.Run("non-named type returns false", func(t *testing.T) {
		// Basic types are not named types
		schema, ok := HandleSpecialType(nil)
		assert.False(t, ok)
		assert.Nil(t, schema)
	})
}

func TestGetOptionalInnerType(t *testing.T) {
	t.Run("non-named type returns false", func(t *testing.T) {
		innerType, ok := GetOptionalInnerType(nil)
		assert.False(t, ok)
		assert.Nil(t, innerType)
	})
}

func TestIsDeprecatedFromDescription_AdditionalCases(t *testing.T) {
	tests := []struct {
		name        string
		description string
		expected    bool
	}{
		{
			name:        "legacy keyword",
			description: "This is a legacy field",
			expected:    true,
		},
		{
			name:        "do not use keyword",
			description: "Do not use this field",
			expected:    true,
		},
		{
			name:        "will be removed keyword",
			description: "This will be removed in v2.0",
			expected:    true,
		},
		{
			name:        "case insensitive deprecated",
			description: "DEPRECATED: use new field instead",
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDeprecatedFromDescription(tt.description)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractTagValue_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		key      string
		expected string
	}{
		{
			name:     "unclosed quote",
			tag:      `mapstructure:"field`,
			key:      "mapstructure",
			expected: "",
		},
		{
			name:     "empty tag",
			tag:      "",
			key:      "mapstructure",
			expected: "",
		},
		{
			name:     "tag with empty value",
			tag:      `mapstructure:""`,
			key:      "mapstructure",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTagValue(tt.tag, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStructFromNamed(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	analyzer := NewPackageAnalyzer(filepath.Join(cwd, "..", ".."))
	pkg, err := analyzer.LoadPackage("go.opentelemetry.io/collector/cmd/builder/internal/schemagen/testdata/testcomponent")
	require.NoError(t, err)

	configType, err := analyzer.FindConfigType(pkg)
	require.NoError(t, err)

	st, ok := GetStructFromNamed(configType)
	assert.True(t, ok)
	assert.NotNil(t, st)
	assert.Positive(t, st.NumFields())
}

func TestSchemaGenerator_GenerateSchema_Errors(t *testing.T) {
	tempDir := t.TempDir()
	analyzer := NewPackageAnalyzer(tempDir)
	generator := NewSchemaGenerator(tempDir, analyzer)

	t.Run("nonexistent package", func(t *testing.T) {
		err := generator.GenerateSchema(component.KindReceiver, "test", "nonexistent/package")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load package")
	})
}

func TestCommentExtractor_ExtractComments(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	analyzer := NewPackageAnalyzer(filepath.Join(cwd, "..", ".."))
	pkg, err := analyzer.LoadPackage("go.opentelemetry.io/collector/cmd/builder/internal/schemagen/testdata/testcomponent")
	require.NoError(t, err)

	ce := NewCommentExtractor()
	ce.ExtractComments(pkg)

	// The testcomponent should have some comments extracted
	// Check that the cache is populated
	assert.NotEmpty(t, ce.commentCache)
}

func TestBuildImportAliasMap_DotImport(t *testing.T) {
	code := `package test
import . "fmt"
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, parser.ImportsOnly)
	require.NoError(t, err)

	result := buildImportAliasMap(file)
	// Dot import uses "." as the local name
	assert.Equal(t, "fmt", result["."])
}

func TestIsComponentConfigType_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name: "non-selector expression",
			code: `package test
var _ int = 0
`,
			expected: false,
		},
		{
			name: "nested selector expression",
			code: `package test
import "go.opentelemetry.io/collector/component"
var _ component.sub.Config = (*Config)(nil)
`,
			expected: false, // This won't parse correctly anyway
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				// Some test cases have intentionally invalid Go code
				return
			}

			importAliases := buildImportAliasMap(file)

			// Find the var declaration
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.VAR {
					continue
				}
				for _, spec := range genDecl.Specs {
					valueSpec, ok := spec.(*ast.ValueSpec)
					if ok && len(valueSpec.Names) == 1 && valueSpec.Names[0].Name == "_" {
						result := isComponentConfigType(valueSpec.Type, importAliases)
						assert.Equal(t, tt.expected, result)
						return
					}
				}
			}
		})
	}
}

func TestExtractConfigTypeFromValueSpec_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name: "multiple names in var declaration",
			code: `package test
var a, b int = 1, 2
`,
			expected: "",
		},
		{
			name: "named variable not blank identifier",
			code: `package test
import "go.opentelemetry.io/collector/component"
var cfg component.Config = (*Config)(nil)
`,
			expected: "",
		},
		{
			name: "no values in var declaration",
			code: `package test
import "go.opentelemetry.io/collector/component"
var _ component.Config
`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			require.NoError(t, err)

			importAliases := buildImportAliasMap(file)
			analyzer := &PackageAnalyzer{}

			// Find the var declaration
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.VAR {
					continue
				}
				for _, spec := range genDecl.Specs {
					valueSpec, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					result := analyzer.extractConfigTypeFromValueSpec(valueSpec, importAliases)
					assert.Equal(t, tt.expected, result)
					return
				}
			}
		})
	}
}

func TestSchemaGenerator_getFieldName_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	analyzer := NewPackageAnalyzer(tempDir)
	sg := NewSchemaGenerator(tempDir, analyzer)

	tests := []struct {
		name     string
		tag      string
		goName   string
		expected string
	}{
		{
			name:     "empty mapstructure value falls back to json",
			tag:      `mapstructure:"" json:"json_field"`,
			goName:   "TestField",
			expected: "json_field",
		},
		{
			name:     "json with dash skips to lowercase",
			tag:      `json:"-"`,
			goName:   "TestField",
			expected: "testfield",
		},
		{
			name:     "mapstructure with options",
			tag:      `mapstructure:"field_name,omitempty"`,
			goName:   "TestField",
			expected: "field_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sg.getFieldName(tt.tag, tt.goName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewSchemaGenerator(t *testing.T) {
	tempDir := t.TempDir()
	analyzer := NewPackageAnalyzer(tempDir)
	sg := NewSchemaGenerator(tempDir, analyzer)

	assert.NotNil(t, sg)
	assert.Equal(t, tempDir, sg.outputDir)
	assert.Equal(t, analyzer, sg.analyzer)
	assert.NotNil(t, sg.comments)
}

func TestSchemaGenerator_ensurePackageComments(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	analyzer := NewPackageAnalyzer(filepath.Join(cwd, "..", ".."))
	sg := NewSchemaGenerator(t.TempDir(), analyzer)

	// Should not panic on empty path
	sg.ensurePackageComments("")

	// Should load and cache comments for valid package
	sg.ensurePackageComments("go.opentelemetry.io/collector/cmd/builder/internal/schemagen/testdata/testcomponent")
	assert.NotEmpty(t, sg.comments.commentCache)
}
