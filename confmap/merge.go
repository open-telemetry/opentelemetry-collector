// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmap // import "go.opentelemetry.io/collector/confmap"

import (
	"reflect"
	"strings"

	"github.com/knadh/koanf/maps"
)

const (
	mergeAllAlias = "*"
)

type mergeOption struct {
	mergePaths []string
}

// WithMergePaths sets an option to merge the lists instead of
// overriding them.
func WithMergePaths(paths []string) MergeOpts {
	return mergeOptionFunc(func(uo *mergeOption) {
		uo.mergePaths = paths
	})
}

type MergeOpts interface {
	apply(*mergeOption)
}

type mergeOptionFunc func(*mergeOption)

func (fn mergeOptionFunc) apply(set *mergeOption) {
	fn(set)
}

func (m *mergeOption) mergeComponentsAppend(src, dest map[string]any) error {
	// mergeComponentsAppend recursively merges the src map into the dest map (left to right),
	// modifying and expanding the dest map in the process.
	// This function does not overwrite lists, and ensures that the final value is a name-aware
	// copy of lists from src and dest.

	// loop through all the paths specified by the user and merge the lists under the specified path
	for _, path := range m.mergePaths {
		merge(path, src, dest)
		if path == mergeAllAlias {
			// every list in the config is now merged, we can exit early.
			break
		}
	}
	// unflatten the new config
	src = maps.Unflatten(src, KeyDelimiter)

	// merge rest of the config.
	maps.Merge(src, dest)
	return nil
}

func merge(path string, src, dest map[string]any) {
	if path == mergeAllAlias {
		// user has specified to merge all the lists in config
		mergeMaps(src, dest)
		return
	}

	pathSplit := strings.Split(path, KeyDelimiter)

	sVal := maps.Search(src, pathSplit)
	dVal := maps.Search(dest, pathSplit)

	srcVal := reflect.ValueOf(sVal)
	destVal := reflect.ValueOf(dVal)

	if srcVal.Kind() != destVal.Kind() {
		// different kinds, maps.Merge will override the old config
		return
	}
	switch destVal.Kind() {
	case reflect.Array, reflect.Slice:
		// both of them are array. Merge the lists

		// delete old value from the src, we'll overwrite the merged value in next step
		maps.Delete(src, pathSplit)

		// Note: path is a delimited by KeyDelimiter. We will call "Unflatten" later to get the final config
		src[path] = mergeSlice(srcVal, destVal)
	case reflect.Map:
		// both of them are maps. Recursively call the mergeMaps
		mergeMaps(sVal.(map[string]any), dVal.(map[string]any))
	}
}

func mergeMaps(src, dest map[string]any) {
	for dKey, dVal := range dest {
		sVal, sOk := src[dKey]
		if !sOk {
			// old key is not present in new config. Hence, add it to src
			src[dKey] = dVal
			continue
		}

		srcVal := reflect.ValueOf(sVal)
		destVal := reflect.ValueOf(dVal)

		if destVal.Kind() != srcVal.Kind() {
			// different kinds, maps.Merge will override the old config
			continue
		}

		switch srcVal.Kind() {
		case reflect.Array, reflect.Slice:
			// both of them are array. Merge them
			src[dKey] = mergeSlice(srcVal, destVal)
		case reflect.Map:
			// both of them are maps. Recursively call the mergeMaps
			mergeMaps(sVal.(map[string]any), dVal.(map[string]any))
		}
	}
}

func mergeSlice(src, dest reflect.Value) any {
	if src.Type() != dest.Type() {
		return src
	}
	slice := reflect.MakeSlice(src.Type(), 0, src.Cap()+dest.Cap())
	for i := 0; i < dest.Len(); i++ {
		slice = reflect.Append(slice, dest.Index(i))
	}

OUTER2:
	for i := 0; i < src.Len(); i++ {
		for j := 0; j < slice.Len(); j++ {
			if slice.Index(j).Equal(src.Index(i)) {
				continue OUTER2
			}
		}
		slice = reflect.Append(slice, src.Index(i))
	}
	return slice.Interface()
}
