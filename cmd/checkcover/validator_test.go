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
}
