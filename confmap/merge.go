package confmap

import (
	"reflect"
)

type MergeFunc func(map[string]any, map[string]any) error

func MergeAppend(src, dest map[string]any) error {
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
			MergeAppend(sVal.(map[string]any), dVal.(map[string]any))
		default:
			// Default case, override the old config
			dest[srcKey] = sVal
		}
	}
	return nil
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
