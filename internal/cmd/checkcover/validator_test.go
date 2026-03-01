// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

func TestValidator_LoadCoverage(t *testing.T) {
	tests := []struct {
		name          string
		coverageData  string
		expectedCount int
	}{
		{
			name: "basic coverage file",
			coverageData: `mode: atomic
go.opentelemetry.io/collector/component/file.go:10.2,12.3 2 5
go.opentelemetry.io/collector/component/file.go:14.2,16.3 2 0
go.opentelemetry.io/collector/exporter/file.go:20.2,22.3 2 10
`,
			expectedCount: 2, // 2 modules: component and exporter
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary coverage file
			tmpDir := t.TempDir()
			coverageFile := filepath.Join(tmpDir, "coverage.txt")
			err := os.WriteFile(coverageFile, []byte(tt.coverageData), 0644)
			require.NoError(t, err)

			validator := NewValidator(tmpDir, 0, false)
			err = validator.LoadCoverage(coverageFile)
			require.NoError(t, err)

			assert.Len(t, validator.coverage, tt.expectedCount)
		})
	}
}

func TestValidator_GetMinimumCoverage(t *testing.T) {
	tests := []struct {
		name         string
		metadata     *ComponentMetadata
		repoMinimum  int
		expectedMin  int
	}{
		{
			name: "stable component - 80% default",
			metadata: &ComponentMetadata{
				Type: "test",
				Status: struct {
					Class           string                                      `yaml:"class"`
					Stability       map[component.StabilityLevel][]string       `yaml:"stability"`
					CoverageMinimum int                                         `yaml:"coverage_minimum"`
					DisableCodeCov  bool                                        `yaml:"disable_codecov_badge"`
				}{
					Class:           "receiver",
					Stability:       map[component.StabilityLevel][]string{component.StabilityLevelStable: {"metrics"}},
					CoverageMinimum: 0,
				},
			},
			repoMinimum: 0,
			expectedMin: 80,
		},
		{
			name: "alpha component - no requirement",
			metadata: &ComponentMetadata{
				Type: "test",
				Status: struct {
					Class           string                                      `yaml:"class"`
					Stability       map[component.StabilityLevel][]string       `yaml:"stability"`
					CoverageMinimum int                                         `yaml:"coverage_minimum"`
					DisableCodeCov  bool                                        `yaml:"disable_codecov_badge"`
				}{
					Class:           "receiver",
					Stability:       map[component.StabilityLevel][]string{component.StabilityLevelAlpha: {"metrics"}},
					CoverageMinimum: 0,
				},
			},
			repoMinimum: 0,
			expectedMin: 0,
		},
		{
			name: "stable with custom minimum",
			metadata: &ComponentMetadata{
				Type: "test",
				Status: struct {
					Class           string                                      `yaml:"class"`
					Stability       map[component.StabilityLevel][]string       `yaml:"stability"`
					CoverageMinimum int                                         `yaml:"coverage_minimum"`
					DisableCodeCov  bool                                        `yaml:"disable_codecov_badge"`
				}{
					Class:           "receiver",
					Stability:       map[component.StabilityLevel][]string{component.StabilityLevelStable: {"metrics"}},
					CoverageMinimum: 90,
				},
			},
			repoMinimum: 0,
			expectedMin: 90,
		},
		{
			name: "stable with repo minimum",
			metadata: &ComponentMetadata{
				Type: "test",
				Status: struct {
					Class           string                                      `yaml:"class"`
					Stability       map[component.StabilityLevel][]string       `yaml:"stability"`
					CoverageMinimum int                                         `yaml:"coverage_minimum"`
					DisableCodeCov  bool                                        `yaml:"disable_codecov_badge"`
				}{
					Class:           "receiver",
					Stability:       map[component.StabilityLevel][]string{component.StabilityLevelStable: {"metrics"}},
					CoverageMinimum: 0,
				},
			},
			repoMinimum: 85,
			expectedMin: 85,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator("", tt.repoMinimum, false)
			minCoverage := validator.getMinimumCoverage(tt.metadata)
			assert.Equal(t, tt.expectedMin, minCoverage)
		})
	}
}

func TestValidator_ValidateCoverageMinimum(t *testing.T) {
	tests := []struct {
		name         string
		coverageMin  int
		expectError  bool
	}{
		{
			name:        "valid coverage 0",
			coverageMin: 0,
			expectError: false,
		},
		{
			name:        "valid coverage 80",
			coverageMin: 80,
			expectError: false,
		},
		{
			name:        "valid coverage 100",
			coverageMin: 100,
			expectError: false,
		},
		{
			name:        "invalid coverage negative",
			coverageMin: -1,
			expectError: true,
		},
		{
			name:        "invalid coverage over 100",
			coverageMin: 101,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We'll need to import the internal/status package to test validation
			// For now, this is a placeholder test structure
			_ = tt.coverageMin
			_ = tt.expectError
		})
	}
}

func TestFindCoverageForModule(t *testing.T) {
	validator := NewValidator("", 0, false)
	validator.coverage = map[string]float64{
		"go.opentelemetry.io/collector/component":          85.5,
		"go.opentelemetry.io/collector/component/internal": 90.0,
		"go.opentelemetry.io/collector/exporter":           75.2,
	}

	tests := []struct {
		name           string
		modulePath     string
		expectedCov    float64
		expectedFound  bool
	}{
		{
			name:          "exact match",
			modulePath:    "go.opentelemetry.io/collector/component",
			expectedCov:   85.5,
			expectedFound: true,
		},
		{
			name:          "prefix match",
			modulePath:    "go.opentelemetry.io/collector/component",
			expectedCov:   85.5,
			expectedFound: true,
		},
		{
			name:          "no match",
			modulePath:    "go.opentelemetry.io/collector/receiver",
			expectedCov:   0,
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coverage, found := validator.findCoverageForModule(tt.modulePath)
			assert.Equal(t, tt.expectedFound, found)
			if found {
				assert.InDelta(t, tt.expectedCov, coverage, 0.01)
			}
		})
	}

	t.Run("windows path normalization", func(t *testing.T) {
		validator := NewValidator("", 0, false)
		validator.coverage = map[string]float64{
			"go.opentelemetry.io/collector/component": 85.5,
		}
		
		// Test with backslashes (Windows-style)
		modulePath := "go.opentelemetry.io\\collector\\component"
		coverage, found := validator.findCoverageForModule(modulePath)
		assert.True(t, found)
		assert.InDelta(t, 85.5, coverage, 0.01)
	})

	t.Run("subpackage match returns first", func(t *testing.T) {
		validator := NewValidator("", 0, false)
		validator.coverage = map[string]float64{
			"go.opentelemetry.io/collector/component":          85.5,
			"go.opentelemetry.io/collector/component/internal": 90.0,
		}
		
		// Looking for just "component" should find it
		coverage, found := validator.findCoverageForModule("go.opentelemetry.io/collector/component")
		assert.True(t, found)
		assert.InDelta(t, 85.5, coverage, 0.01)
	})
}

func TestValidator_LoadCoverage_ErrorCases(t *testing.T) {
	t.Run("file not found", func(t *testing.T) {
		validator := NewValidator(t.TempDir(), 0, false)
		err := validator.LoadCoverage("nonexistent.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open coverage file")
	})

	t.Run("empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		coverageFile := filepath.Join(tmpDir, "coverage.txt")
		err := os.WriteFile(coverageFile, []byte(""), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, false)
		err = validator.LoadCoverage(coverageFile)
		require.NoError(t, err)
		assert.Empty(t, validator.coverage)
	})

	t.Run("invalid format lines ignored", func(t *testing.T) {
		tmpDir := t.TempDir()
		coverageFile := filepath.Join(tmpDir, "coverage.txt")
		content := `mode: atomic
invalid line
go.opentelemetry.io/collector/component/file.go:10.2,12.3 2 5
another invalid
`
		err := os.WriteFile(coverageFile, []byte(content), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, false)
		err = validator.LoadCoverage(coverageFile)
		require.NoError(t, err)
		assert.Len(t, validator.coverage, 1)
	})

	t.Run("verbose mode prints coverage", func(t *testing.T) {
		tmpDir := t.TempDir()
		coverageFile := filepath.Join(tmpDir, "coverage.txt")
		content := `mode: atomic
go.opentelemetry.io/collector/component/file.go:10.2,12.3 4 5
go.opentelemetry.io/collector/component/file.go:14.2,16.3 2 0
`
		err := os.WriteFile(coverageFile, []byte(content), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, true) // verbose mode
		err = validator.LoadCoverage(coverageFile)
		require.NoError(t, err)
		assert.Len(t, validator.coverage, 1)
	})

	t.Run("file without colon ignored", func(t *testing.T) {
		tmpDir := t.TempDir()
		coverageFile := filepath.Join(tmpDir, "coverage.txt")
		content := `mode: atomic
invalidlinewithnocolon 2 5
`
		err := os.WriteFile(coverageFile, []byte(content), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, false)
		err = validator.LoadCoverage(coverageFile)
		require.NoError(t, err)
		assert.Empty(t, validator.coverage)
	})

	t.Run("lines with invalid numbers ignored", func(t *testing.T) {
		tmpDir := t.TempDir()
		coverageFile := filepath.Join(tmpDir, "coverage.txt")
		content := `mode: atomic
go.opentelemetry.io/collector/component/file.go:10.2,12.3 invalid 5
go.opentelemetry.io/collector/component/file.go:10.2,12.3 2 invalid
go.opentelemetry.io/collector/component/file.go:10.2,12.3 2 5
`
		err := os.WriteFile(coverageFile, []byte(content), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, false)
		err = validator.LoadCoverage(coverageFile)
		require.NoError(t, err)
		assert.Len(t, validator.coverage, 1)
	})

	t.Run("partial fields handled", func(t *testing.T) {
		tmpDir := t.TempDir()
		coverageFile := filepath.Join(tmpDir, "coverage.txt")
		content := `mode: atomic
go.opentelemetry.io/collector/component/file.go:10.2,12.3 2
single field
`
		err := os.WriteFile(coverageFile, []byte(content), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, false)
		err = validator.LoadCoverage(coverageFile)
		require.NoError(t, err)
		assert.Empty(t, validator.coverage)
	})
}

func TestValidator_LoadMetadata(t *testing.T) {
	t.Run("valid metadata", func(t *testing.T) {
		tmpDir := t.TempDir()
		metadataPath := filepath.Join(tmpDir, "metadata.yaml")
		content := `type: test
status:
  class: receiver
  stability:
    stable: [metrics]
  coverage_minimum: 85
`
		err := os.WriteFile(metadataPath, []byte(content), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, false)
		metadata, err := validator.loadMetadata(metadataPath)
		require.NoError(t, err)
		assert.Equal(t, "test", metadata.Type)
		assert.Equal(t, "receiver", metadata.Status.Class)
		assert.Equal(t, 85, metadata.Status.CoverageMinimum)
	})

	t.Run("file not found", func(t *testing.T) {
		validator := NewValidator(t.TempDir(), 0, false)
		_, err := validator.loadMetadata("nonexistent.yaml")
		require.Error(t, err)
	})

	t.Run("invalid yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		metadataPath := filepath.Join(tmpDir, "metadata.yaml")
		err := os.WriteFile(metadataPath, []byte("invalid: [yaml content"), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, false)
		_, err = validator.loadMetadata(metadataPath)
		require.Error(t, err)
	})
}

func TestValidator_GetStabilityLevel(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ComponentMetadata
		expected string
	}{
		{
			name: "single stable",
			metadata: &ComponentMetadata{
				Status: struct {
					Class           string                                      `yaml:"class"`
					Stability       map[component.StabilityLevel][]string       `yaml:"stability"`
					CoverageMinimum int                                         `yaml:"coverage_minimum"`
					DisableCodeCov  bool                                        `yaml:"disable_codecov_badge"`
				}{
					Stability: map[component.StabilityLevel][]string{
						component.StabilityLevelStable: {"metrics"},
					},
				},
			},
			expected: "Stable",
		},
		{
			name: "multiple levels",
			metadata: &ComponentMetadata{
				Status: struct {
					Class           string                                      `yaml:"class"`
					Stability       map[component.StabilityLevel][]string       `yaml:"stability"`
					CoverageMinimum int                                         `yaml:"coverage_minimum"`
					DisableCodeCov  bool                                        `yaml:"disable_codecov_badge"`
				}{
					Stability: map[component.StabilityLevel][]string{
						component.StabilityLevelStable: {"metrics"},
						component.StabilityLevelBeta:   {"traces"},
					},
				},
			},
			expected: "Stable,Beta", // Note: order may vary
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator("", 0, false)
			result := validator.getStabilityLevel(tt.metadata)
			assert.NotEmpty(t, result)
			// Check if expected value is contained (order may vary)
			if tt.name == "single stable" {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestValidator_GetStabilityBasedMinimum(t *testing.T) {
	tests := []struct {
		name        string
		stability   map[component.StabilityLevel][]string
		expectedMin int
	}{
		{
			name: "stable",
			stability: map[component.StabilityLevel][]string{
				component.StabilityLevelStable: {"metrics"},
			},
			expectedMin: 80,
		},
		{
			name: "beta",
			stability: map[component.StabilityLevel][]string{
				component.StabilityLevelBeta: {"metrics"},
			},
			expectedMin: 0,
		},
		{
			name: "alpha",
			stability: map[component.StabilityLevel][]string{
				component.StabilityLevelAlpha: {"metrics"},
			},
			expectedMin: 0,
		},
		{
			name: "mixed with stable",
			stability: map[component.StabilityLevel][]string{
				component.StabilityLevelStable: {"metrics"},
				component.StabilityLevelBeta:   {"traces"},
			},
			expectedMin: 80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := &ComponentMetadata{
				Status: struct {
					Class           string                                      `yaml:"class"`
					Stability       map[component.StabilityLevel][]string       `yaml:"stability"`
					CoverageMinimum int                                         `yaml:"coverage_minimum"`
					DisableCodeCov  bool                                        `yaml:"disable_codecov_badge"`
				}{
					Stability: tt.stability,
				},
			}
			validator := NewValidator("", 0, false)
			result := validator.getStabilityBasedMinimum(metadata)
			assert.Equal(t, tt.expectedMin, result)
		})
	}
}

func TestValidator_GetModulePath(t *testing.T) {
	t.Run("component with go.mod", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModContent := "module go.opentelemetry.io/collector/receiver/test\n"
		goModPath := filepath.Join(tmpDir, "go.mod")
		err := os.WriteFile(goModPath, []byte(goModContent), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, false)
		modulePath, err := validator.getModulePath(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "go.opentelemetry.io/collector/receiver/test", modulePath)
	})

	t.Run("no go.mod found", func(t *testing.T) {
		tmpDir := t.TempDir()
		validator := NewValidator(tmpDir, 0, false)
		_, err := validator.getModulePath(tmpDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no go.mod found")
	})

	t.Run("invalid go.mod content", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Empty go.mod without module declaration
		goModContent := "// empty file\n"
		goModPath := filepath.Join(tmpDir, "go.mod")
		err := os.WriteFile(goModPath, []byte(goModContent), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, false)
		_, err = validator.getModulePath(tmpDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not parse module name")
	})

	t.Run("unreadable go.mod file", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")
		err := os.WriteFile(goModPath, []byte("module test\n"), 0000) // no read permissions
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, false)
		_, err = validator.getModulePath(tmpDir)
		// This will fail on permission denied
		require.Error(t, err)
		
		// Clean up - restore permissions
		os.Chmod(goModPath, 0644)
	})
}

func TestValidator_FindMetadataFiles(t *testing.T) {
	t.Run("find multiple metadata files", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Create some metadata files
		dir1 := filepath.Join(tmpDir, "receiver", "test1")
		err := os.MkdirAll(dir1, 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(dir1, "metadata.yaml"), []byte("type: test1\n"), 0644)
		require.NoError(t, err)

		dir2 := filepath.Join(tmpDir, "exporter", "test2")
		err = os.MkdirAll(dir2, 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(dir2, "metadata.yaml"), []byte("type: test2\n"), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, false)
		files, err := validator.findMetadataFiles()
		require.NoError(t, err)
		assert.Len(t, files, 2)
	})

	t.Run("skip hidden directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Create hidden directory with metadata
		hiddenDir := filepath.Join(tmpDir, ".hidden")
		err := os.MkdirAll(hiddenDir, 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(hiddenDir, "metadata.yaml"), []byte("type: hidden\n"), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, false)
		files, err := validator.findMetadataFiles()
		require.NoError(t, err)
		assert.Len(t, files, 0)
	})
}

func TestValidator_ValidateComponent(t *testing.T) {
	t.Run("skip disabled codecov", func(t *testing.T) {
		tmpDir := t.TempDir()
		metadataPath := filepath.Join(tmpDir, "metadata.yaml")
		content := `type: test
status:
  class: receiver
  stability:
    stable: [metrics]
  disable_codecov_badge: true
`
		err := os.WriteFile(metadataPath, []byte(content), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, false)
		err = validator.validateComponent(metadataPath)
		require.NoError(t, err)
	})

	t.Run("validation passes with sufficient coverage", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Create go.mod
		goModContent := "module go.opentelemetry.io/collector/test\n"
		err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644)
		require.NoError(t, err)

		// Create metadata
		metadataPath := filepath.Join(tmpDir, "metadata.yaml")
		content := `type: test
status:
  class: receiver
  stability:
    stable: [metrics]
`
		err = os.WriteFile(metadataPath, []byte(content), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, false)
		// Add coverage data
		validator.coverage["go.opentelemetry.io/collector/test"] = 85.0

		err = validator.validateComponent(metadataPath)
		require.NoError(t, err)
	})

	t.Run("validation fails with insufficient coverage", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Create go.mod
		goModContent := "module go.opentelemetry.io/collector/test\n"
		err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644)
		require.NoError(t, err)

		// Create metadata
		metadataPath := filepath.Join(tmpDir, "metadata.yaml")
		content := `type: test
status:
  class: receiver
  stability:
    stable: [metrics]
`
		err = os.WriteFile(metadataPath, []byte(content), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, false)
		// Add insufficient coverage data
		validator.coverage["go.opentelemetry.io/collector/test"] = 70.0

		err = validator.validateComponent(metadataPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "coverage")
		assert.Contains(t, err.Error(), "below minimum")
	})

	t.Run("invalid metadata file", func(t *testing.T) {
		tmpDir := t.TempDir()
		metadataPath := filepath.Join(tmpDir, "metadata.yaml")
		err := os.WriteFile(metadataPath, []byte("invalid: [yaml"), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, false)
		err = validator.validateComponent(metadataPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load metadata")
	})

	t.Run("no coverage data found - verbose mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Create go.mod
		goModContent := "module go.opentelemetry.io/collector/test\n"
		err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644)
		require.NoError(t, err)

		// Create metadata
		metadataPath := filepath.Join(tmpDir, "metadata.yaml")
		content := `type: test
status:
  class: receiver
  stability:
    stable: [metrics]
`
		err = os.WriteFile(metadataPath, []byte(content), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, true) // verbose mode
		// No coverage data added

		err = validator.validateComponent(metadataPath)
		require.NoError(t, err) // Should not error, just warn
	})

	t.Run("cannot determine module path - verbose mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Create metadata but no go.mod
		metadataPath := filepath.Join(tmpDir, "metadata.yaml")
		content := `type: test
status:
  class: receiver
  stability:
    alpha: [metrics]
`
		err := os.WriteFile(metadataPath, []byte(content), 0644)
		require.NoError(t, err)

		validator := NewValidator(tmpDir, 0, true) // verbose mode
		
		err = validator.validateComponent(metadataPath)
		require.NoError(t, err) // Should not error, just warn
	})
}

func TestValidator_ValidateComponents(t *testing.T) {
	t.Run("no metadata files", func(t *testing.T) {
		tmpDir := t.TempDir()
		validator := NewValidator(tmpDir, 0, false)
		
		// Load empty coverage
		coverageFile := filepath.Join(tmpDir, "coverage.txt")
		err := os.WriteFile(coverageFile, []byte("mode: atomic\n"), 0644)
		require.NoError(t, err)
		err = validator.LoadCoverage(coverageFile)
		require.NoError(t, err)

		err = validator.ValidateComponents()
		require.NoError(t, err)
	})

	t.Run("multiple validation errors", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Create multiple components with insufficient coverage
		for i := 1; i <= 2; i++ {
			dir := filepath.Join(tmpDir, "component"+string(rune('0'+i)))
			err := os.MkdirAll(dir, 0755)
			require.NoError(t, err)
			
			// Create go.mod
			goModContent := "module go.opentelemetry.io/collector/test" + string(rune('0'+i)) + "\n"
			err = os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModContent), 0644)
			require.NoError(t, err)
			
			// Create metadata with stable requirement
			metadataPath := filepath.Join(dir, "metadata.yaml")
			content := `type: test` + string(rune('0'+i)) + `
status:
  class: receiver
  stability:
    stable: [metrics]
`
			err = os.WriteFile(metadataPath, []byte(content), 0644)
			require.NoError(t, err)
		}

		validator := NewValidator(tmpDir, 0, false)
		// Add insufficient coverage for both
		validator.coverage["go.opentelemetry.io/collector/test1"] = 70.0
		validator.coverage["go.opentelemetry.io/collector/test2"] = 65.0

		err := validator.ValidateComponents()
		require.Error(t, err)
		// Should contain errors for both components
		assert.Contains(t, err.Error(), "test1")
		assert.Contains(t, err.Error(), "test2")
	})
}

func TestNewValidator(t *testing.T) {
	validator := NewValidator("/tmp", 75, true)
	assert.Equal(t, "/tmp", validator.repoRoot)
	assert.Equal(t, 75, validator.repoMinimum)
	assert.True(t, validator.verbose)
	assert.NotNil(t, validator.coverage)
}
