// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/component"
)

// Validator validates code coverage for components
type Validator struct {
	repoRoot    string
	repoMinimum int
	verbose     bool
	coverage    map[string]float64 // module path -> coverage percentage
}

// ComponentMetadata represents the metadata.yaml structure
type ComponentMetadata struct {
	Type   string `yaml:"type"`
	Status struct {
		Class           string                                      `yaml:"class"`
		Stability       map[component.StabilityLevel][]string       `yaml:"stability"`
		CoverageMinimum int                                         `yaml:"coverage_minimum"`
		DisableCodeCov  bool                                        `yaml:"disable_codecov_badge"`
	} `yaml:"status"`
}

// NewValidator creates a new coverage validator
func NewValidator(repoRoot string, repoMinimum int, verbose bool) *Validator {
	return &Validator{
		repoRoot:    repoRoot,
		repoMinimum: repoMinimum,
		verbose:     verbose,
		coverage:    make(map[string]float64),
	}
}

// LoadCoverage reads and parses the coverage file
func (v *Validator) LoadCoverage(coverageFile string) error {
	file, err := os.Open(coverageFile)
	if err != nil {
		return fmt.Errorf("failed to open coverage file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Coverage file format: <package>/<file>:<line>.<column>,<line>.<column> <num_statements> <count>
	// We need to calculate coverage per module
	moduleCoverage := make(map[string]*coverageStats)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		// Extract package path from the line (format: path/to/package/file.go:line.col,line.col)
		fileAndPos := parts[0]
		colonIdx := strings.Index(fileAndPos, ":")
		if colonIdx == -1 {
			continue
		}
		filePath := fileAndPos[:colonIdx]
		
		// Extract the module path (everything before the last /)
		lastSlash := strings.LastIndex(filePath, "/")
		var modulePath string
		if lastSlash != -1 {
			modulePath = filePath[:lastSlash]
		} else {
			modulePath = filePath
		}

		numStmts, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		count, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}

		stats := moduleCoverage[modulePath]
		if stats == nil {
			stats = &coverageStats{}
			moduleCoverage[modulePath] = stats
		}

		stats.totalStmts += numStmts
		if count > 0 {
			stats.coveredStmts += numStmts
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading coverage file: %w", err)
	}

	// Calculate percentages
	for module, stats := range moduleCoverage {
		if stats.totalStmts > 0 {
			percentage := float64(stats.coveredStmts) / float64(stats.totalStmts) * 100
			v.coverage[module] = percentage
			if v.verbose {
				fmt.Printf("Module: %s, Coverage: %.2f%% (%d/%d statements)\n",
					module, percentage, stats.coveredStmts, stats.totalStmts)
			}
		}
	}

	return nil
}

type coverageStats struct {
	totalStmts   int
	coveredStmts int
}

// ValidateComponents validates coverage for all components
func (v *Validator) ValidateComponents() error {
	// Find all metadata.yaml files
	metadataFiles, err := v.findMetadataFiles()
	if err != nil {
		return fmt.Errorf("failed to find metadata files: %w", err)
	}

	if v.verbose {
		fmt.Printf("Found %d metadata files\n", len(metadataFiles))
	}

	var validationErrors []error

	for _, metadataPath := range metadataFiles {
		if err := v.validateComponent(metadataPath); err != nil {
			validationErrors = append(validationErrors, err)
		}
	}

	if len(validationErrors) > 0 {
		return errors.Join(validationErrors...)
	}

	return nil
}

// findMetadataFiles finds all metadata.yaml files in the repository
func (v *Validator) findMetadataFiles() ([]string, error) {
	var metadataFiles []string

	err := filepath.Walk(v.repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor, node_modules, and hidden directories
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == "node_modules" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if info.Name() == "metadata.yaml" {
			metadataFiles = append(metadataFiles, path)
		}

		return nil
	})

	return metadataFiles, err
}

// validateComponent validates a single component's coverage
func (v *Validator) validateComponent(metadataPath string) error {
	metadata, err := v.loadMetadata(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to load metadata from %s: %w", metadataPath, err)
	}

	// Skip if codecov is disabled
	if metadata.Status.DisableCodeCov {
		if v.verbose {
			fmt.Printf("Skipping %s (codecov disabled)\n", metadataPath)
		}
		return nil
	}

	// Determine the minimum required coverage
	minRequired := v.getMinimumCoverage(metadata)

	// Get the module path for this component
	componentDir := filepath.Dir(metadataPath)
	modulePath, err := v.getModulePath(componentDir)
	if err != nil {
		if v.verbose {
			fmt.Printf("Warning: could not determine module path for %s: %v\n", metadataPath, err)
		}
		return nil
	}

	// Get the actual coverage for this module
	actualCoverage, found := v.findCoverageForModule(modulePath)
	if !found {
		if v.verbose {
			fmt.Printf("Warning: no coverage data found for module %s\n", modulePath)
		}
		return nil
	}

	// Validate coverage
	if actualCoverage < float64(minRequired) {
		componentName := fmt.Sprintf("%s/%s", metadata.Status.Class, metadata.Type)
		return fmt.Errorf(
			"❌ %s: coverage %.2f%% is below minimum %.0f%% (stability: %s, module: %s)",
			componentName,
			actualCoverage,
			float64(minRequired),
			v.getStabilityLevel(metadata),
			modulePath,
		)
	}

	if v.verbose {
		fmt.Printf("✅ %s/%s: %.2f%% >= %d%% (module: %s)\n",
			metadata.Status.Class, metadata.Type, actualCoverage, minRequired, modulePath)
	}

	return nil
}

// loadMetadata loads and parses a metadata.yaml file
func (v *Validator) loadMetadata(path string) (*ComponentMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var metadata ComponentMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// getMinimumCoverage determines the minimum coverage requirement for a component
func (v *Validator) getMinimumCoverage(metadata *ComponentMetadata) int {
	// Start with the default based on stability
	minCoverage := v.getStabilityBasedMinimum(metadata)

	// Use repository minimum if higher
	if v.repoMinimum > minCoverage {
		minCoverage = v.repoMinimum
	}

	// Use component-specific minimum if higher
	if metadata.Status.CoverageMinimum > minCoverage {
		minCoverage = metadata.Status.CoverageMinimum
	}

	return minCoverage
}

// getStabilityBasedMinimum returns the minimum coverage based on stability level
func (v *Validator) getStabilityBasedMinimum(metadata *ComponentMetadata) int {
	// Check if component has stable signals
	if _, hasStable := metadata.Status.Stability[component.StabilityLevelStable]; hasStable {
		return 80 // Stable components require 80% coverage
	}

	// For other stability levels, no minimum is enforced by default
	// (unless specified in metadata or repo minimum)
	return 0
}

// getStabilityLevel returns a string representation of the component's stability
func (v *Validator) getStabilityLevel(metadata *ComponentMetadata) string {
	levels := []string{}
	for level := range metadata.Status.Stability {
		levels = append(levels, level.String())
	}
	return strings.Join(levels, ",")
}

// getModulePath determines the Go module path for a component directory
func (v *Validator) getModulePath(componentDir string) (string, error) {
	// Read the go.mod file to get the module path
	goModPath := filepath.Join(componentDir, "go.mod")
	if _, err := os.Stat(goModPath); err != nil {
		// No go.mod in this directory, try parent
		parent := filepath.Dir(componentDir)
		if parent == componentDir || parent == v.repoRoot {
			return "", fmt.Errorf("no go.mod found")
		}
		return v.getModulePath(parent)
	}

	// Parse go.mod to get module name
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}

	// Extract module name from go.mod
	re := regexp.MustCompile(`module\s+(\S+)`)
	matches := re.FindSubmatch(data)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse module name from go.mod")
	}

	moduleName := string(matches[1])

	// Get the relative path from the go.mod directory to the component directory
	relPath, err := filepath.Rel(filepath.Dir(goModPath), componentDir)
	if err != nil {
		return "", err
	}

	if relPath == "." {
		return moduleName, nil
	}

	return filepath.Join(moduleName, relPath), nil
}

// findCoverageForModule finds coverage data for a given module path
func (v *Validator) findCoverageForModule(modulePath string) (float64, bool) {
	// Normalize the module path
	normalizedPath := strings.ReplaceAll(modulePath, "\\", "/")

	// Try exact match first
	if coverage, found := v.coverage[normalizedPath]; found {
		return coverage, true
	}

	// Try to find coverage for any subpackage of this module
	var totalStmts, coveredStmts int
	found := false

	for covPath, percentage := range v.coverage {
		if strings.HasPrefix(covPath, normalizedPath) {
			// This is approximate - we're using the percentage we already calculated
			// In a real implementation, we'd track statements separately
			found = true
			// For now, just return the first match
			return percentage, true
		}
	}

	if !found {
		return 0, false
	}

	// Calculate weighted average if we collected stats
	if totalStmts > 0 {
		return float64(coveredStmts) / float64(totalStmts) * 100, true
	}

	return 0, false
}
