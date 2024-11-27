// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	startComponentList = `# Start Collector components list`
	endComponentList   = `# End Collector components list`
)

func folderToShortName(folder string) string {
	if folder == "internal/coreinternal" {
		return "internal/core"
	}
	path := strings.Split(folder, "/")
	switch path[0] {
	case "receiver", "exporter", "extension", "processor", "connector":
		path[1] = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(path[1], "internal"), "extension"), "exporter"), "connector"), "processor"), "receiver")
		path[len(path)-1] = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(path[len(path)-1], "internal"), "extension"), "exporter"), "connector"), "processor"), "receiver")
	default:
	}

	return strings.Join(path, "/")
}

type issueTemplatesGenerator struct{}

func (itg issueTemplatesGenerator) generate(data *githubData) error {
	keys := map[string]struct{}{}
	for _, f := range data.folders {
		keys[folderToShortName(f)] = struct{}{}
	}
	shortNames := make([]string, 0, len(keys))
	for k := range keys {
		shortNames = append(shortNames, k)
	}
	sort.Strings(shortNames)
	replacement := []byte(startComponentList + "\n      - " + strings.Join(shortNames, "\n      - ") + "\n      " + endComponentList)
	issuesFolder := filepath.Join(".github", "ISSUE_TEMPLATE")
	entries, err := os.ReadDir(issuesFolder)
	if err != nil {
		return err
	}
	for _, e := range entries {
		templateContents, err := os.ReadFile(filepath.Join(issuesFolder, e.Name())) // nolint: gosec
		if err != nil {
			return err
		}
		matchOldContent := regexp.MustCompile("(?s)" + startComponentList + ".*" + endComponentList)
		oldContent := matchOldContent.FindSubmatch(templateContents)
		if len(oldContent) > 0 {
			templateContents = bytes.ReplaceAll(templateContents, oldContent[0], replacement)
			err = os.WriteFile(filepath.Join(issuesFolder, e.Name()), templateContents, 0o600)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
