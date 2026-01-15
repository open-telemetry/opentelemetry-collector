// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/confmap/internal"

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/knadh/koanf/maps"
	"go.yaml.in/yaml/v3"
)

type MergeOptions struct {
	mode string
}

func newOptions(mode string) *MergeOptions {
	return &MergeOptions{
		mode: mode,
	}
}

func FetchMergePaths(yamlBytes []byte) (map[string]*MergeOptions, error) {
	// fetchMergePaths takes the input yaml and extracts the path that has custom tags set
	// It returns a "map" of paths->options, where options are the merge options to use.
	// Right now, we only support list merging options. In future, we can support an option to override maps.
	var root yaml.Node
	if err := yaml.Unmarshal(yamlBytes, &root); err != nil {
		return nil, fmt.Errorf("error unmarshalling yaml: %w", err)
	}
	m := map[string]*MergeOptions{}
	if err := walkYAML(nil, &root, m, []string{}); err != nil {
		return nil, fmt.Errorf("failed to walk the yaml tree: %w", err)
	}
	return m, nil
}

func walkYAML(key, node *yaml.Node, res map[string]*MergeOptions, path []string) error {
	// walkYAML recursively walks through the yaml tree and populates "res" with paths to merge.
	// It keeps track of current path in "path" array. This is needed for final merge operation
	if key != nil {
		path = append(path, key.Value)
	}
	switch node.Kind {
	case yaml.DocumentNode:
		for _, n := range node.Content {
			if err := walkYAML(nil, n, res, path); err != nil {
				return err
			}
		}

	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			if err := walkYAML(node.Content[i], node.Content[i+1], res, path); err != nil {
				return err
			}
		}

	case yaml.SequenceNode:
		for _, n := range node.Content {
			s := node.Tag
			// By default, a yaml sequence node will have a "!!seq" tag.
			// check if it has a custom tag and extract merge mode and other options.
			if s != "!!seq" {
				s = strings.TrimPrefix(s, "!")
				q, err := url.ParseQuery(s)
				if err != nil {
					return fmt.Errorf("failed to parse tag %v at path %v: %w", node.Tag, path, err)
				}
				if err := validateTag(q); err != nil {
					return fmt.Errorf("failed to validate tag %v at path %v: %w", node.Tag, path, err)
				}
				res[strings.Join(path, "::")] = newOptions(q.Get("mode"))
				if err := walkYAML(nil, n, res, path); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func mergeAppend(mergeOpts map[string]*MergeOptions) func(src, dest map[string]any) error {
	// mergeOpts is the list of paths where component lists should be merged.

	return func(src, dest map[string]any) error {
		// mergeAppend recursively merges the src map into the dest map (left to right),
		// modifying and expanding the dest map in the process.
		// This function does not overwrite component lists, and ensures that the
		// final value is a name-aware copy of lists from src and dest.

		// Flatten both source and destination maps
		srcFlat, _ := maps.Flatten(src, []string{}, KeyDelimiter)
		destFlat, _ := maps.Flatten(dest, []string{}, KeyDelimiter)

		for sKey, sVal := range srcFlat {
			opt := isMatch(sKey, mergeOpts)
			if opt == nil {
				// no option found for this path. Continue
				continue
			}

			dVal, dOk := destFlat[sKey]
			if !dOk {
				continue // Let maps.Merge handle missing keys
			}

			srcVal := reflect.ValueOf(sVal)
			destVal := reflect.ValueOf(dVal)

			if srcVal.Kind() != destVal.Kind() {
				// If user has specified different types for the same key, continue and let maps.Merge handle this
				// User shouldn't really be doing this, but this protects against any panics we can face in reflect
				continue
			}

			if opt.mode == "append" {
				// Only merge if the value is a slice or array; let maps.Merge handle other types
				if srcVal.Kind() == reflect.Slice || srcVal.Kind() == reflect.Array {
					srcFlat[sKey] = mergeSlice(srcVal, destVal)
				}
			}
		}

		// Unflatten and merge
		mergedSrc := maps.Unflatten(srcFlat, KeyDelimiter)
		maps.Merge(mergedSrc, dest)

		return nil
	}
}

// isMatch checks if a key matches any of the extracted paths
func isMatch(sKey string, mergeOpts map[string]*MergeOptions) *MergeOptions {
	for key := range mergeOpts {
		if strings.EqualFold(key, sKey) {
			return mergeOpts[key]
		}
	}
	return nil
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

func isPresent(slice, val reflect.Value) bool {
	for i := 0; i < slice.Len(); i++ {
		if reflect.DeepEqual(val.Interface(), slice.Index(i).Interface()) {
			return true
		}
	}
	return false
}

func validateTag(query url.Values) error {
	switch query.Get("mode") {
	case "append":
	default:
		return fmt.Errorf("invalid mode specified: %v", query.Get("mode"))
	}
	return nil
}
