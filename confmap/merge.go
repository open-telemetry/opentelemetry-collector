// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmap // import "go.opentelemetry.io/collector/confmap"

import (
	"reflect"

	"github.com/knadh/koanf/maps"
)

func mergeAppend(src, dest map[string]any) error {
	// mergeAppend recursively merges the src map into the dest map (left to right),
	// modifying and expanding the dest map in the process.
	// This function does not overwrite lists, and ensures that the final value is a name-aware
	// copy of lists from src and dest.

	// First, merge the src and dest config maps
	mergeMaps(src, dest)

	// Second, unflatten the new config
	src = maps.Unflatten(src, KeyDelimiter)

	// merge rest of the config.
	maps.Merge(src, dest)
	return nil
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
