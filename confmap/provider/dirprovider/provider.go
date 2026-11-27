// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

package dirprovider // import "go.opentelemetry.io/collector/confmap/provider/dirprovider"

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/collector/confmap"
)

const schemeName = "dir"

type provider struct{}

// NewFactory returns a factory for a confmap.Provider that reads and merges
// configuration from the files in a directory or matching a glob pattern.
//
// This Provider supports the "dir" scheme. The value after "dir:" is a directory
// path or a glob pattern:
//
//	dir:/folder            - every *.yaml and *.yml directly in /folder (one level)
//	dir:/folder/*.yaml     - single-level glob match
//	dir:/folder/**/*.yaml  - recursive glob match
//
// Matched files are merged in lexicographical order of their absolute paths.
// Merging uses the same semantics as the rest of the Collector, including the
// confmap.enableMergeAppendOption feature gate.
func NewFactory() confmap.ProviderFactory {
	return confmap.NewProviderFactory(newProvider)
}

func newProvider(confmap.ProviderSettings) confmap.Provider {
	return &provider{}
}

func (*provider) Retrieve(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, schemeName+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, schemeName)
	}

	pattern := uri[len(schemeName)+1:]

	files, err := resolveFiles(pattern)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no configuration files found for %q", uri)
	}

	merged := confmap.New()
	for _, f := range files {
		content, err := os.ReadFile(filepath.Clean(f))
		if err != nil {
			return nil, fmt.Errorf("unable to read the file %q: %w", f, err)
		}
		ret, err := confmap.NewRetrievedFromYAML(content)
		if err != nil {
			return nil, fmt.Errorf("unable to parse the file %q: %w", f, err)
		}
		conf, err := ret.AsConf()
		if err != nil {
			return nil, fmt.Errorf("unable to read the file %q as config: %w", f, err)
		}
		if err := merged.Merge(conf); err != nil {
			return nil, fmt.Errorf("unable to merge the file %q: %w", f, err)
		}
	}

	return confmap.NewRetrieved(merged.ToStringMap())
}

// resolveFiles returns the sorted list of config files for the given pattern.
// A bare directory (no glob metacharacters) matches *.yaml and *.yml one level deep.
func resolveFiles(pattern string) ([]string, error) {
	if !hasGlobMeta(pattern) {
		info, err := os.Stat(pattern)
		if err != nil {
			return nil, fmt.Errorf("unable to stat %q: %w", pattern, err)
		}
		if !info.IsDir() {
			if !isYAML(pattern) {
				return nil, fmt.Errorf("%q is not a YAML file; dir: provider only loads .yaml/.yml files", pattern)
			}
			return []string{pattern}, nil
		}
		var files []string
		entries, err := os.ReadDir(pattern)
		if err != nil {
			return nil, fmt.Errorf("unable to read directory %q: %w", pattern, err)
		}
		for _, e := range entries {
			if !e.Type().IsRegular() {
				continue
			}
			if isYAML(e.Name()) {
				files = append(files, filepath.Join(pattern, e.Name()))
			}
		}
		return files, nil
	}
	if !strings.Contains(pattern, "**") {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}
		files, err := filterRegularFiles(matches)
		if err != nil {
			return nil, err
		}
		return files, nil
	}
	return recursiveGlob(pattern)
}

// recursiveGlob resolves a pattern containing "**" using filepath.WalkDir.
// The pattern is split at the first "**": the literal part before it is the
// directory to walk, and the part after it (typically "*.yaml") is matched
// against the base name of each file found at any depth. An empty tail (a
// pattern ending in "**") matches every file under the root.
//
// Only files at least one directory level below root are matched, consistent
// with common shell globstar semantics where "**" requires at least one
// intermediate path component. Files directly in root are skipped.
//
// Only the first "**" is significant and the tail is matched against each
// file's base name, so a pattern with "**" in the middle of the path
// (e.g. "a/**/b/*.yaml") is not supported and will match nothing. A pattern
// that begins with "**" walks the current working directory.
func recursiveGlob(pattern string) ([]string, error) {
	root, rest, _ := strings.Cut(pattern, "**")
	tail := strings.TrimLeft(rest, `/\`)
	if tail == "" {
		// A pattern ending in "**" (e.g. dir:/folder/**) means "every file
		// under root"; an empty tail would otherwise match nothing.
		tail = "*"
	}
	if root == "" {
		root = "."
	}
	root = filepath.Clean(root)

	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		// Skip files directly in root — "**" requires at least one subdirectory.
		if filepath.Dir(path) == root {
			return nil
		}
		ok, matchErr := filepath.Match(tail, filepath.Base(path))
		if matchErr != nil {
			return fmt.Errorf("invalid glob pattern %q: %w", pattern, matchErr)
		}
		if ok {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("unable to walk %q for pattern %q: %w", root, pattern, err)
	}
	return files, nil
}

// filterRegularFiles drops directories (e.g. matched by a broad glob) from the
// given paths. A failure to stat a matched path is returned as an error, since
// filepath.Glob only returns paths that existed at scan time, so a stat failure
// indicates an abnormal condition (e.g. a permission error or a broken symlink)
// rather than a routine skip.
func filterRegularFiles(paths []string) ([]string, error) {
	var out []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("unable to stat %q: %w", p, err)
		}
		if info.IsDir() {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func hasGlobMeta(p string) bool {
	return strings.ContainsAny(p, "*?[")
}

func isYAML(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml" || ext == ".yml"
}

func (*provider) Scheme() string {
	return schemeName
}

func (*provider) Shutdown(context.Context) error {
	return nil
}
