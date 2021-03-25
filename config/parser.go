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

// package config implements the configuration Parser.
package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

const (
	// KeyDelimiter is used as the default key delimiter in the default viper instance.
	KeyDelimiter = "::"
)

// NewViper creates a new Viper instance with key delimiter KeyDelimiter instead of the
// default ".". This way configs can have keys that contain ".".
func NewViper() *viper.Viper {
	return viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
}

// NewParser creates a new Parser instance.
func NewParser() *Parser {
	return &Parser{
		v: NewViper(),
	}
}

// NewParserFromFile creates a new Parser by reading the given file.
func NewParserFromFile(fileName string) (*Parser, error) {
	// Read yaml config from file
	v := NewViper()
	v.SetConfigFile(fileName)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("unable to read the file %v: %w", fileName, err)
	}
	return ParserFromViper(v), nil
}

// ParserFromViper creates a Parser from a Viper instance.
func ParserFromViper(v *viper.Viper) *Parser {
	return &Parser{v: v}
}

// Parser loads configuration.
type Parser struct {
	v *viper.Viper
}

// AllKeys returns all keys holding a value, regardless of where they are set.
// Nested keys are returned with a KeyDelimiter separator
func (l *Parser) AllKeys() []string {
	return l.v.AllKeys()
}

// Unmarshal unmarshals the config into a struct. Make sure that the tags
// on the fields of the structure are properly set.
func (l *Parser) Unmarshal(rawVal interface{}) error {
	return l.v.Unmarshal(rawVal)
}

// UnmarshalExact unmarshals the config into a struct, erroring if a field is nonexistent.
func (l *Parser) UnmarshalExact(intoCfg interface{}) error {
	l.v.AllKeys()
	return l.v.UnmarshalExact(intoCfg)
}

// Get can retrieve any value given the key to use.
func (l *Parser) Get(key string) interface{} {
	return l.v.Get(key)
}

// Set sets the value for the key.
func (l *Parser) Set(key string, value interface{}) {
	l.v.Set(key, value)
}

// Sub returns new Parser instance representing a sub tree of this instance.
func (l *Parser) Sub(key string) (*Parser, error) {
	// Copied from the Viper but changed to use the same delimiter
	// and return error if the sub is not a map.
	// See https://github.com/spf13/viper/issues/871
	data := l.Get(key)
	if data == nil {
		return NewParser(), nil
	}

	if reflect.TypeOf(data).Kind() == reflect.Map {
		subv := NewViper()
		// Cannot return error because the subv is empty.
		_ = subv.MergeConfigMap(cast.ToStringMap(data))
		return ParserFromViper(subv), nil
	}

	return nil, fmt.Errorf("unexpected sub-config value kind for key:%s value:%v kind:%v)", key, data, reflect.TypeOf(data).Kind())
}

// deepSearch scans deep maps, following the key indexes listed in the
// sequence "path".
// The last value is expected to be another map, and is returned.
//
// In case intermediate keys do not exist, or map to a non-map value,
// a new map is created and inserted, and the search continues from there:
// the initial map "m" may be modified!
// This function comes from Viper code https://github.com/spf13/viper/blob/5253694/util.go#L201-L230
// It is used here because of https://github.com/spf13/viper/issues/819
func deepSearch(m map[string]interface{}, path []string) map[string]interface{} {
	for _, k := range path {
		m2, ok := m[k]
		if !ok {
			// intermediate key does not exist
			// => create it and continue from there
			m3 := make(map[string]interface{})
			m[k] = m3
			m = m3
			continue
		}
		m3, ok := m2.(map[string]interface{})
		if !ok {
			// intermediate key is a value
			// => replace with a new map
			m3 = make(map[string]interface{})
			m[k] = m3
		}
		// continue search from here
		m = m3
	}
	return m
}

// ToStringMap creates a map[string]interface{} from a Parser.
func (l *Parser) ToStringMap() map[string]interface{} {
	// This is equivalent to l.v.AllSettings() but it maps nil values
	// We can't use AllSettings here because of https://github.com/spf13/viper/issues/819

	m := map[string]interface{}{}
	// start from the list of keys, and construct the map one value at a time
	for _, k := range l.v.AllKeys() {
		value := l.v.Get(k)
		path := strings.Split(k, KeyDelimiter)
		lastKey := strings.ToLower(path[len(path)-1])
		deepestMap := deepSearch(m, path[0:len(path)-1])
		// set innermost value
		deepestMap[lastKey] = value
	}
	return m
}
