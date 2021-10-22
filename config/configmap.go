// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config // import "go.opentelemetry.io/collector/config"

import (
	"fmt"
	"io"
	"io/ioutil"
	"reflect"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cast"
)

const (
	// KeyDelimiter is used as the default key delimiter in the default koanf instance.
	KeyDelimiter = "::"
)

// NewMap creates a new empty config.Map instance.
func NewMap() *Map {
	return &Map{k: koanf.New(KeyDelimiter)}
}

// NewMapFromFile creates a new config.Map by reading the given file.
func NewMapFromFile(fileName string) (*Map, error) {
	// Read yaml config from file.
	p := NewMap()
	if err := p.k.Load(file.Provider(fileName), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("unable to read the file %v: %w", fileName, err)
	}
	return p, nil
}

// NewMapFromBuffer creates a new config.Map by reading the given yaml buffer.
func NewMapFromBuffer(buf io.Reader) (*Map, error) {
	content, err := ioutil.ReadAll(buf)
	if err != nil {
		return nil, err
	}

	p := NewMap()
	if err := p.k.Load(rawbytes.Provider(content), yaml.Parser()); err != nil {
		return nil, err
	}

	return p, nil
}

// NewMapFromStringMap creates a config.Map from a map[string]interface{}.
func NewMapFromStringMap(data map[string]interface{}) *Map {
	p := NewMap()
	// Cannot return error because the koanf instance is empty.
	_ = p.k.Load(confmap.Provider(data, KeyDelimiter), nil)
	return p
}

// Map represents the raw configuration map for the OpenTelemetry Collector.
// The config.Map can be unmarshalled into the Collector's config using the "configunmarshaler" package.
type Map struct {
	k *koanf.Koanf
}

// AllKeys returns all keys holding a value, regardless of where they are set.
// Nested keys are returned with a KeyDelimiter separator.
func (l *Map) AllKeys() []string {
	return l.k.Keys()
}

// Unmarshal unmarshalls the config into a struct.
// Tags on the fields of the structure must be properly set.
func (l *Map) Unmarshal(rawVal interface{}) error {
	decoder, err := mapstructure.NewDecoder(decoderConfig(rawVal))
	if err != nil {
		return err
	}
	return decoder.Decode(l.ToStringMap())
}

// UnmarshalExact unmarshalls the config into a struct, erroring if a field is nonexistent.
func (l *Map) UnmarshalExact(rawVal interface{}) error {
	dc := decoderConfig(rawVal)
	dc.ErrorUnused = true
	decoder, err := mapstructure.NewDecoder(dc)
	if err != nil {
		return err
	}
	return decoder.Decode(l.ToStringMap())
}

// Get can retrieve any value given the key to use.
func (l *Map) Get(key string) interface{} {
	return l.k.Get(key)
}

// Set sets the value for the key.
func (l *Map) Set(key string, value interface{}) {
	// koanf doesn't offer a direct setting mechanism so merging is required.
	merged := koanf.New(KeyDelimiter)
	_ = merged.Load(confmap.Provider(map[string]interface{}{key: value}, KeyDelimiter), nil)
	l.k.Merge(merged)
}

// IsSet checks to see if the key has been set in any of the data locations.
// IsSet is case-insensitive for a key.
func (l *Map) IsSet(key string) bool {
	return l.k.Exists(key)
}

// Merge merges the input given configuration into the existing config.
// Note that the given map may be modified.
func (l *Map) Merge(in *Map) error {
	return l.k.Merge(in.k)
}

// Sub returns new Parser instance representing a sub-config of this instance.
// It returns an error is the sub-config is not a map (use Get()) and an empty Parser if
// none exists.
func (l *Map) Sub(key string) (*Map, error) {
	data := l.Get(key)
	if data == nil {
		return NewMap(), nil
	}

	if reflect.TypeOf(data).Kind() == reflect.Map {
		return NewMapFromStringMap(cast.ToStringMap(data)), nil
	}

	return nil, fmt.Errorf("unexpected sub-config value kind for key:%s value:%v kind:%v)", key, data, reflect.TypeOf(data).Kind())
}

// ToStringMap creates a map[string]interface{} from a Parser.
func (l *Map) ToStringMap() map[string]interface{} {
	return maps.Unflatten(l.k.All(), KeyDelimiter)
}

// decoderConfig returns a default mapstructure.DecoderConfig capable of parsing time.Duration
// and weakly converting config field values to primitive types.  It also ensures that maps
// whose values are nil pointer structs resolved to the zero value of the target struct (see
// expandNilStructPointers). A decoder created from this mapstructure.DecoderConfig will decode
// its contents to the result argument.
func decoderConfig(result interface{}) *mapstructure.DecoderConfig {
	return &mapstructure.DecoderConfig{
		Result:           result,
		Metadata:         nil,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			expandNilStructPointersFunc,
			stringToSliceHookFunc,
			stringMapToComponentIDMapHookFunc,
			stringToTimeDurationHookFunc,
			textUnmarshallerHookFunc,
		),
	}
}

var (
	stringToSliceHookFunc        = mapstructure.StringToSliceHookFunc(",")
	stringToTimeDurationHookFunc = mapstructure.StringToTimeDurationHookFunc()
	textUnmarshallerHookFunc     = mapstructure.TextUnmarshallerHookFunc()

	componentIDType = reflect.TypeOf(NewComponentID("foo"))
	sentinel        = &struct{}{}
)

// In cases where a config has a mapping of something to a struct pointers
// we want nil values to resolve to a pointer to the zero value of the
// underlying struct just as we want nil values of a mapping of something
// to a struct to resolve to the zero value of that struct.
//
// e.g. given a config type:
// type Config struct { Struct *SomeStruct `mapstructure:"struct"` }
//
// and yaml of:
// config:
//   struct:
//
// we want an unmarshaled Config to be equivalent to
// Config{Struct: &SomeStruct{}} instead of Config{Struct: nil}
var expandNilStructPointersFunc = func(from reflect.Value, to reflect.Value) (interface{}, error) {
	if to.Kind() == reflect.Ptr {
		if to.IsNil() {
			to.Set(reflect.New(to.Type().Elem()))
		}
	}
	toIsStruct := to.Kind() == reflect.Struct || (to.Kind() == reflect.Ptr && to.Elem().Kind() == reflect.Struct)
	fmt.Printf("toKind=%v\n", to.Kind())
	fmt.Printf("toType=%v\n", to.Type())
	if to.Kind() == reflect.Ptr {
		fmt.Printf("toIsNil=%v\n", to.IsNil())
		fmt.Printf("toElemKind=%v\n", to.Elem().Type().Kind())
	}
	fmt.Printf("toIsStruct=%v\n", toIsStruct)
	if from.Interface() == sentinel {
		// Zero only if nil pointer.
		if to.Kind() == reflect.Ptr && to.IsNil() {
			to.Set(reflect.Zero(to.Type()))
		}
		// If the destination is a struct, then return am empty map[string]interface.
		if toIsStruct {
			return make(map[string]interface{}), nil
		}
		fmt.Printf("ret=%v\n", reflect.Zero(to.Type()).Interface())
		// Replace the source to a zero value struct of the same type as the destination.
		return reflect.Zero(to.Type()).Interface(), nil
	}
	// If any map has an element with a nil value, replace that with a "sentinel" if the destination is a struct, or an
	// empty map[string]interface if the destination is a map[string]*Struct. The reason to use a "sentinel" is that we
	// should have initialized that with a zero value of the same type as the destination, but destination could be a
	// struct pointer or a map and in case of a struct we would need to duplicate the code that looks up for the
	// "mapstructure" tag to identify the field.
	toIsMapToStruct := to.Kind() == reflect.Map && to.Type().Elem().Kind() == reflect.Ptr && to.Type().Elem().Elem().Kind() == reflect.Struct
	//fmt.Printf("toIsMapToStruct=%v\n", toIsMapToStruct)
	if from.Kind() == reflect.Map && (toIsStruct || toIsMapToStruct) {
		fromRange := from.MapRange()
		for fromRange.Next() {
			fromKey := fromRange.Key()
			fromValue := fromRange.Value()
			// ensure that we've run into a nil pointer instance
			if fromValue.IsNil() {
				if toIsStruct {
					from.SetMapIndex(fromKey, reflect.ValueOf(sentinel))
				} else {
					from.SetMapIndex(fromKey, reflect.ValueOf(make(map[string]interface{})))
				}
			}
		}
	}
	return from.Interface(), nil
}

// stringMapToComponentIDMapHookFunc returns a DecodeHookFunc that converts a map[string]interface{} to
// map[ComponentID]interface{}.
var stringMapToComponentIDMapHookFunc = func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f.Kind() != reflect.Map || f.Key().Kind() != reflect.String {
		return data, nil
	}

	if t.Kind() != reflect.Map || t.Key() != componentIDType {
		return data, nil
	}

	m := make(map[ComponentID]interface{})
	for k, v := range data.(map[string]interface{}) {
		id, err := NewComponentIDFromString(k)
		if err != nil {
			return nil, err
		}
		if _, ok := m[id]; ok {
			return nil, fmt.Errorf("duplicate name %q after trimming spaces %v", k, id)
		}
		m[id] = v
	}
	return m, nil
}
