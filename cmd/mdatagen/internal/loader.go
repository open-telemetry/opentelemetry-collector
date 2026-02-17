// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
)

const componentLabelsRelPath = ".github/component_labels.txt"

var (
	packageNameFunc                 = packageName
	applyComponentLabelFromRepoFunc = applyComponentLabelFromRepo
)

func setAttributeDefaultFields(attrs map[AttributeName]Attribute) {
	for k, v := range attrs {
		v.FullName = k
		if v.RequirementLevel == "" {
			v.RequirementLevel = AttributeRequirementLevelRecommended
		}
		attrs[k] = v
	}
}

type TemplateContext struct {
	Metadata
	// Package name for generated code.
	Package string
}

func LoadMetadata(filePath string) (Metadata, error) {
	cp, err := fileprovider.NewFactory().Create(confmaptest.NewNopProviderSettings()).Retrieve(context.Background(), "file:"+filePath, nil)
	if err != nil {
		return Metadata{}, err
	}

	conf, err := cp.AsConf()
	if err != nil {
		return Metadata{}, err
	}

	md := Metadata{ShortFolderName: shortFolderName(filePath), Tests: Tests{Host: "newMdatagenNopHost()"}}
	err = conf.Unmarshal(&md)
	if err != nil {
		return md, err
	}
	packageName, err := packageNameFunc(filepath.Dir(filePath))
	if err != nil {
		return md, fmt.Errorf("unable to determine package name: %w", err)
	}
	md.PackageName = packageName

	if md.ScopeName == "" {
		md.ScopeName = packageName
	}
	if md.GeneratedPackageName == "" {
		md.GeneratedPackageName = "metadata"
	}

	if err := md.Validate(); err != nil {
		return md, err
	}

	applyComponentLabelFromRepoFunc(filePath, &md)

	if md.Label == "" && md.Status != nil {
		md.Label = fmt.Sprintf("%s/%s", md.Status.Class, md.ShortFolderName)
	}

	setAttributeDefaultFields(md.Attributes)
	setAttributeDefaultFields(md.ResourceAttributes)

	return md, nil
}

var componentTypes = []string{
	"connector",
	"exporter",
	"extension",
	"processor",
	"scraper",
	"receiver",
}

func shortFolderName(filePath string) string {
	parentFolder := filepath.Base(filepath.Dir(filePath))
	for _, cType := range componentTypes {
		if before, ok := strings.CutSuffix(parentFolder, cType); ok {
			return before
		}
	}
	return parentFolder
}

func packageName(filePath string) (string, error) {
	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}")
	cmd.Dir = filePath
	output, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", fmt.Errorf("unable to determine package name: %v failed: (stderr) %v", cmd.Args, string(ee.Stderr))
		}

		return "", fmt.Errorf("unable to determine package name: %v failed: %v %w", cmd.Args, string(output), err)
	}
	return strings.TrimSpace(string(output)), nil
}

func findRepoRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, componentLabelsRelPath)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repo root not found from %s", start)
		}
		dir = parent
	}
}

// loadComponentLabels loads .github/component_labels.txt into a map[repoPath]label.
// repoPath -> label (second column)
func loadComponentLabels(path string) (map[string]string, error) {
	// Validate path is within expected location
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("invalid path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	labels := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		repoPath := parts[0] // e.g. "extension/storage/filestorage"
		label := parts[1]    // e.g. "extension/storage/filestorage"
		labels[repoPath] = label
	}
	return labels, nil
}

func applyComponentLabelFromRepo(filePath string, md *Metadata) {
	// Find the repo root (directory that has .github/component_labels.txt)
	repoRoot, err := findRepoRoot(filepath.Dir(filePath))
	if err != nil {
		return
	}

	// Load labels file
	labelsPath := filepath.Join(repoRoot, componentLabelsRelPath)
	labelMap, err := loadComponentLabels(labelsPath)
	if err != nil {
		return
	}

	// Determine repo-relative directory for this component
	compDir := filepath.Dir(filePath)
	rel, err := filepath.Rel(repoRoot, compDir)
	if err != nil {
		return
	}
	rel = filepath.ToSlash(rel)

	// If there is a mapping, set the label. If not, we just leave md.Label empty.
	if lbl, ok := labelMap[rel]; ok {
		md.Label = lbl
	}
}
