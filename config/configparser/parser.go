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

package configparser

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

// NewParser creates a new empty Parser instance.
func NewParser() *ConfigMap {
	return &ConfigMap{k: koanf.New(KeyDelimiter)}
}

// NewParserFromFile creates a new Parser by reading the given file.
func NewParserFromFile(fileName string) (*ConfigMap, error) {
	// Read yaml config from file.
	p := NewParser()
	if err := p.k.Load(file.Provider(fileName), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("unable to read the file %v: %w", fileName, err)
	}
	return p, nil
}

// NewParserFromBuffer creates a new Parser by reading the given yaml buffer.
func NewParserFromBuffer(buf io.Reader) (*ConfigMap, error) {
	content, err := ioutil.ReadAll(buf)
	if err != nil {
		return nil, err
	}

	p := NewParser()
	if err := p.k.Load(rawbytes.Provider(content), yaml.Parser()); err != nil {
		return nil, err
	}

	return p, nil
}

// NewParserFromStringMap creates a parser from a map[string]interface{}.
func NewParserFromStringMap(data map[string]interface{}) *ConfigMap {
	p := NewParser()
	// Cannot return error because the koanf instance is empty.
	_ = p.k.Load(confmap.Provider(data, KeyDelimiter), nil)
	return p
}

// ConfigMap loads configuration.
type ConfigMap struct {
	k *koanf.Koanf
}

// AllKeys returns all keys holding a value, regardless of where they are set.
// Nested keys are returned with a KeyDelimiter separator.
func (l *ConfigMap) AllKeys() []string {
	return l.k.Keys()
}

// Unmarshal unmarshalls the config into a struct.
// Tags on the fields of the structure must be properly set.
func (l *ConfigMap) Unmarshal(rawVal interface{}) error {
	decoder, err := mapstructure.NewDecoder(decoderConfig(rawVal))
	if err != nil {
		return err
	}
	return decoder.Decode(l.ToStringMap())
}

// UnmarshalExact unmarshalls the config into a struct, erroring if a field is nonexistent.
func (l *ConfigMap) UnmarshalExact(intoCfg interface{}) error {
	dc := decoderConfig(intoCfg)
	dc.ErrorUnused = true
	decoder, err := mapstructure.NewDecoder(dc)
	if err != nil {
		return err
	}
	return decoder.Decode(l.ToStringMap())
}

// Get can retrieve any value given the key to use.
func (l *ConfigMap) Get(key string) interface{} {
	return l.k.Get(key)
}

// Set sets the value for the key.
func (l *ConfigMap) Set(key string, value interface{}) {
	// koanf doesn't offer a direct setting mechanism so merging is required.
	merged := koanf.New(KeyDelimiter)
	merged.Load(confmap.Provider(map[string]interface{}{key: value}, KeyDelimiter), nil)
	l.k.Merge(merged)
}

// IsSet checks to see if the key has been set in any of the data locations.
// IsSet is case-insensitive for a key.
func (l *ConfigMap) IsSet(key string) bool {
	return l.k.Exists(key)
}

// MergeStringMap merges the configuration from the given map with the existing config.
// Note that the given map may be modified.
func (l *ConfigMap) MergeStringMap(cfg map[string]interface{}) error {
	toMerge := koanf.New(KeyDelimiter)
	toMerge.Load(confmap.Provider(cfg, KeyDelimiter), nil)
	return l.k.Merge(toMerge)
}

// Sub returns new Parser instance representing a sub-config of this instance.
// It returns an error is the sub-config is not a map (use Get()) and an empty Parser if
// none exists.
func (l *ConfigMap) Sub(key string) (*ConfigMap, error) {
	data := l.Get(key)
	if data == nil {
		return NewParser(), nil
	}

	if reflect.TypeOf(data).Kind() == reflect.Map {
		subParser := NewParser()
		// Cannot return error because the subv is empty.
		_ = subParser.MergeStringMap(cast.ToStringMap(data))
		return subParser, nil
	}

	return nil, fmt.Errorf("unexpected sub-config value kind for key:%s value:%v kind:%v)", key, data, reflect.TypeOf(data).Kind())
}

// ToStringMap creates a map[string]interface{} from a Parser.
func (l *ConfigMap) ToStringMap() map[string]interface{} {
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
			expandNilStructPointers(),
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	}
}

// In cases where a config has a mapping of something to a struct pointers
// we want nil values to resolve to a pointer to the zero value of the
// underlying struct just as we want nil values of a mapping of something
// to a struct to resolve to the zero value of that struct.
//
// e.g. given a config type:
// type Config struct { Thing *SomeStruct `mapstructure:"thing"` }
//
// and yaml of:
// config:
//   thing:
//
// we want an unmarshalled Config to be equivalent to
// Config{Thing: &SomeStruct{}} instead of Config{Thing: nil}
func expandNilStructPointers() mapstructure.DecodeHookFunc {
	return func(from reflect.Value, to reflect.Value) (interface{}, error) {
		// ensure we are dealing with map to map comparison
		if from.Kind() == reflect.Map && to.Kind() == reflect.Map {
			toElem := to.Type().Elem()
			// ensure that map values are pointers to a struct
			// (that may be nil and require manual setting w/ zero value)
			if toElem.Kind() == reflect.Ptr && toElem.Elem().Kind() == reflect.Struct {
				fromRange := from.MapRange()
				for fromRange.Next() {
					fromKey := fromRange.Key()
					fromValue := fromRange.Value()
					// ensure that we've run into a nil pointer instance
					if fromValue.IsNil() {
						newFromValue := reflect.New(toElem.Elem())
						from.SetMapIndex(fromKey, newFromValue)
					}
				}
			}
		}
		return from.Interface(), nil
	}
}
