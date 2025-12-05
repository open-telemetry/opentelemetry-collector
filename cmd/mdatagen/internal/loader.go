// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
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
	packageName, err := packageName(filepath.Dir(filePath))
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
