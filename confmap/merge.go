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

type mergeComponents struct {
	mergePaths []string
}

func (m *mergeComponents) mergeComponentsAppend(src, dest map[string]any) error {
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
		// different kinds, override the old config
		src[path] = sVal
		return
	}
	switch destVal.Kind() {
	case reflect.Array, reflect.Slice:
		// both of them are array. Merge the lists

		// delete old value from the src, we'll overwrite the merged value in next step
		maps.Delete(src, pathSplit)

		// Note: path is a delimited by KeyDelimiter. We will call "Unflatten" later to get the final config
		src[path] = mergeSlice(destVal, srcVal)
	case reflect.Map:
		// both of them are maps. Recursively call the mergeMaps
		mergeMaps(sVal.(map[string]any), dVal.(map[string]any))
	default:
		// Default case, override the old config
		src[path] = sVal
	}
}

func mergeMaps(new, old map[string]any) {
	for oldKey, oVal := range old {
		nVal, newOk := new[oldKey]
		if !newOk {
			// old key is not present in new config, hence, add it
			new[oldKey] = nVal
			continue
		}

		newVal := reflect.ValueOf(nVal)
		oldVal := reflect.ValueOf(oVal)

		if newVal.Kind() != oldVal.Kind() {
			// different kinds, override the old config
			new[oldKey] = nVal
			continue
		}

		switch oldVal.Kind() {
		case reflect.Array, reflect.Slice:
			// both of them are array. Merge them
			new[oldKey] = mergeSlice(oldVal, newVal)
		case reflect.Map:
			// both of them are maps. Recursively call the mergeMaps
			mergeMaps(nVal.(map[string]any), oVal.(map[string]any))
		default:
			// Default case, override the old config
			new[oldKey] = nVal
		}
	}
}

func mergeSlice(old, new reflect.Value) any {
	if old.Type() != new.Type() {
		return new
	}
	slice := reflect.MakeSlice(old.Type(), 0, old.Cap()+new.Cap())
	for i := 0; i < old.Len(); i++ {
		slice = reflect.Append(slice, old.Index(i))
	}

OUTER2:
	for i := 0; i < new.Len(); i++ {
		for j := 0; j < slice.Len(); j++ {
			if slice.Index(j).Equal(new.Index(i)) {
				continue OUTER2
			}
		}
		slice = reflect.Append(slice, new.Index(i))
	}
	return slice.Interface()
}
