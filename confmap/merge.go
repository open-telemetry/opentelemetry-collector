// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmap // import "go.opentelemetry.io/collector/confmap"

import (
	"net/url"
	"reflect"
	"strings"

	"github.com/gobwas/glob"
	"github.com/knadh/koanf/maps"
	"go.yaml.in/yaml/v3"
)

type options struct {
	mode       string
	duplicates bool

	// glob to be set after the path is compiled
	glob glob.Glob
}

func newOptions(mode string, duplicates bool) *options {
	if mode == "" {
		mode = "append"
	}
	return &options{
		mode:       mode,
		duplicates: duplicates,
	}
}

func fetchMergePaths(yamlBytes []byte) map[string]*options {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yamlBytes), &root); err != nil {
		panic(err)
	}
	m := map[string]*options{}
	walkYAML(nil, &root, m, []string{})
	return m
}

func walkYAML(key *yaml.Node, node *yaml.Node, res map[string]*options, path []string) {
	// walkYAML recursively walks through the yaml tree and populates "res" with paths to merge.
	// It keeps track of current path in "path" array. This is needed for final merge operation
	if key != nil {
		path = append(path, key.Value)
	}
	switch node.Kind {
	case yaml.DocumentNode:
		for _, n := range node.Content {
			walkYAML(nil, n, res, path)
		}

	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			walkYAML(node.Content[i], node.Content[i+1], res, path)
		}

	case yaml.SequenceNode:
		for _, n := range node.Content {
			s := node.Tag
			// By default, a yaml sequence node will have a "!!seq" tag.
			// check if it has a custom tag and extract merge mode and other options.
			if s != "!!seq" {
				s = strings.TrimPrefix(s, "!")
				q, _ := url.ParseQuery(s)

				res[strings.Join(path, "::")] = newOptions(q.Get("mode"), q.Get("duplicates") == "true")
				walkYAML(nil, n, res, path)
			}
		}
	}
}

func mergeAppend(mergeOpts map[string]*options) func(src, dest map[string]any) error {
	// mergeOpts is the list of paths where component lists should be merged.

	return func(src, dest map[string]any) error {
		// mergeAppend recursively merges the src map into the dest map (left to right),
		// modifying and expanding the dest map in the process.
		// This function does not overwrite component lists, and ensures that the
		// final value is a name-aware copy of lists from src and dest.

		// Compile the globs once
		for path, opt := range mergeOpts {
			if g, err := glob.Compile(path); err == nil {
				opt.glob = g
			}
		}

		// Flatten both source and destination maps
		srcFlat, _ := maps.Flatten(src, []string{}, KeyDelimiter)
		destFlat, _ := maps.Flatten(dest, []string{}, KeyDelimiter)

		for sKey, sVal := range srcFlat {
			if !isMatch(sKey, mergeOpts) {
				continue
			}

			dVal, dOk := destFlat[sKey]
			if !dOk {
				continue // Let maps.Merge handle missing keys
			}

			srcVal := reflect.ValueOf(sVal)
			destVal := reflect.ValueOf(dVal)

			// Only merge if the value is a slice or array; let maps.Merge handle other types
			if srcVal.Kind() == reflect.Slice || srcVal.Kind() == reflect.Array {
				srcFlat[sKey] = mergeSlice(srcVal, destVal)
			}
		}

		// Unflatten and merge
		mergedSrc := maps.Unflatten(srcFlat, KeyDelimiter)
		maps.Merge(mergedSrc, dest)

		return nil
	}
}

// isMatch checks if a key matches any glob in the list
func isMatch(key string, mergeOpts map[string]*options) bool {
	for _, opt := range mergeOpts {
		if opt.glob.Match(key) {
			return true
		}
	}
	return false
}

func mergeSlice(src, dest reflect.Value) any {
	slice := reflect.MakeSlice(src.Type(), 0, src.Cap()+dest.Cap())
	for i := 0; i < dest.Len(); i++ {
		slice = reflect.Append(slice, dest.Index(i))
	}

	for i := 0; i < src.Len(); i++ {
		if isPresent(slice, src.Index(i)) {
			continue
		}
		slice = reflect.Append(slice, src.Index(i))
	}
	return slice.Interface()
}

func isPresent(slice reflect.Value, val reflect.Value) bool {
	for i := 0; i < slice.Len(); i++ {
		if reflect.DeepEqual(val.Interface(), slice.Index(i).Interface()) {
			return true
		}
	}
	return false
}
