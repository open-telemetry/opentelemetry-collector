package confmap

import (
	"reflect"

	"github.com/knadh/koanf/maps"
)

type MergeFunc func(map[string]any, map[string]any) error

func MergeComponentsAppend(new, old map[string]any) error {
	newService := maps.Search(new, []string{"service"})
	oldService := maps.Search(old, []string{"service"})
	if oldSer, ok := oldService.(map[string]any); ok {
		if newSer, ok := newService.(map[string]any); ok {
			mergeServices(newSer, oldSer)
			new["service"] = oldSer
		}
	}
	maps.Merge(new, old)
	return nil
}

func mergeServices(src, dest map[string]any) {
	for srcKey, sVal := range src {
		dVal, destOk := dest[srcKey]
		if !destOk {
			dest[srcKey] = sVal
			continue
		}

		destVal := reflect.ValueOf(dVal)
		srcVal := reflect.ValueOf(sVal)

		if destVal.Kind() != srcVal.Kind() {
			// different kinds, override the old config
			dest[srcKey] = sVal
			continue
		}

		switch srcVal.Kind() {
		case reflect.Array, reflect.Slice:
			// both of them are array. Merge them
			dest[srcKey] = mergeSlice(destVal, srcVal)
		case reflect.Map:
			// both of them are maps. Recursively call the MergeAppend
			mergeServices(sVal.(map[string]any), dVal.(map[string]any))
		default:
			// Default case, override the old config
			dest[srcKey] = sVal
		}
	}
}

func mergeSlice(src, dest reflect.Value) any {
	if src.Type() != dest.Type() {
		return dest
	}
	slice := reflect.MakeSlice(src.Type(), 0, src.Cap()+dest.Cap())
	for i := 0; i < src.Len(); i++ {
		slice = reflect.Append(slice, src.Index(i))
	}

OUTER2:
	for i := 0; i < dest.Len(); i++ {
		for j := 0; j < slice.Len(); j++ {
			if slice.Index(j).Equal(dest.Index(i)) {
				continue OUTER2
			}
		}
		slice = reflect.Append(slice, dest.Index(i))
	}
	return slice.Interface()
}
