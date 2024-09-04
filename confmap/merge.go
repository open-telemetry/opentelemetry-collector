// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmap // import "go.opentelemetry.io/collector/confmap"

import (
	"reflect"

	"github.com/knadh/koanf/maps"
)

type MergeFunc func(map[string]any, map[string]any) error

func mergeComponentsAppend(new, old map[string]any) error {
	newService := maps.Search(new, []string{"service"})
	oldService := maps.Search(old, []string{"service"})
	if oldSer, ok := oldService.(map[string]any); ok {
		if newSer, ok := newService.(map[string]any); ok {
			mergeServices(newSer, oldSer)
			// override the `service` in new config.
			new["service"] = newSer
		}
	}
	// merge rest of the config.
	maps.Merge(new, old)
	return nil
}

func mergeServices(new, old map[string]any) {
	for oldKey, oVal := range old {
		nVal, newOk := new[oldKey]
		if !newOk {
			new[oldKey] = oVal
			continue
		}

		newVal := reflect.ValueOf(nVal)
		oldVal := reflect.ValueOf(oVal)

		if newVal.Kind() != oldVal.Kind() {
			// different kinds, override the old config
			new[oldKey] = oVal
			continue
		}

		switch oldVal.Kind() {
		case reflect.Array, reflect.Slice:
			// both of them are array. Merge them
			new[oldKey] = mergeSlice(oldVal, newVal)
		case reflect.Map:
			// both of them are maps. Recursively call the MergeAppend
			mergeServices(nVal.(map[string]any), oVal.(map[string]any))
		default:
			// Default case, override the old config
			new[oldKey] = oVal
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
