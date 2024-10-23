// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
)

func setAttributesFullName(attrs map[AttributeName]Attribute) {
	for k, v := range attrs {
		v.FullName = k
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

	md := Metadata{ShortFolderName: shortFolderName(filePath), Tests: Tests{Host: "componenttest.NewNopHost()"}}
	if err = conf.Unmarshal(&md); err != nil {
		return md, err
	}
	if md.ScopeName == "" {
		md.ScopeName, err = packageName()
		if err != nil {
			return md, err
		}
	}
	if md.GeneratedPackageName == "" {
		md.GeneratedPackageName = "metadata"
	}

	if err = md.Validate(); err != nil {
		return md, err
	}

	setAttributesFullName(md.Attributes)
	setAttributesFullName(md.ResourceAttributes)

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
		if strings.HasSuffix(parentFolder, cType) {
			return strings.TrimSuffix(parentFolder, cType)
		}
	}
	return parentFolder
}

func packageName() (string, error) {
	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
