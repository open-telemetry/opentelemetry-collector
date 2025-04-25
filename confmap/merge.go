// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmap // import "go.opentelemetry.io/collector/confmap"

import (
	"reflect"

	"github.com/gobwas/glob"
	"github.com/knadh/koanf/maps"
)

func mergeAppend(src, dest map[string]any) error {
	// Compile the globs once
	patterns := []string{
		"service::extensions",
		"service::**::receivers",
		"service::**::exporters",
	}
	var globs []glob.Glob
	for _, p := range patterns {
		if g, err := glob.Compile(p); err == nil {
			globs = append(globs, g)
		}
	}

	// Flatten both source and destination maps
	srcFlat, _ := maps.Flatten(src, []string{}, KeyDelimiter)
	destFlat, _ := maps.Flatten(dest, []string{}, KeyDelimiter)

	for key, sVal := range srcFlat {
		if !isMatch(key, globs) {
			continue
		}

		dVal, exists := destFlat[key]
		if !exists {
			continue // Let maps.Merge handle missing keys
		}

		srcVal := reflect.ValueOf(sVal)
		destVal := reflect.ValueOf(dVal)

		// Only merge if the value is a slice or array; let maps.Merge handle other types
		if srcVal.Kind() == reflect.Slice || srcVal.Kind() == reflect.Array {
			srcFlat[key] = mergeSlice(srcVal, destVal)
		}
	}

	// Unflatten and merge
	mergedSrc := maps.Unflatten(srcFlat, KeyDelimiter)
	maps.Merge(mergedSrc, dest)

	return nil
}

// isMatch checks if a key matches any glob in the list
func isMatch(key string, globs []glob.Glob) bool {
	for _, g := range globs {
		if g.Match(key) {
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
		if slice.Index(i).Equal(val) {
			return true
		}
	}
	return false
}
