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

package viper

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

const (
	// ViperDelimiter is used as the default key delimiter in the default viper instance
	ViperDelimiter = "::"
)

var defaultOptions = []viper.DecoderConfigOption{viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
	mapstructure.StringToTimeDurationHookFunc(),
	mapstructure.StringToSliceHookFunc(","),
	humanStringToBytes,
))}

type Viper struct {
	*viper.Viper
}

func (v *Viper) UnmarshalKey(key string, rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	return v.Viper.UnmarshalKey(key, rawVal, append(defaultOptions, opts...)...)
}

func (v *Viper) Unmarshal(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	return v.Viper.Unmarshal(rawVal, append(defaultOptions, opts...)...)
}

func (v *Viper) UnmarshalExact(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	return v.Viper.UnmarshalExact(rawVal, append(defaultOptions, opts...)...)
}

func (v *Viper) Sub(key string) *Viper {
	return &Viper{Viper: v.Viper.Sub(key)}
}

// Copied from the Viper but changed to use the same delimiter
// and return error if the sub is not a map.
// See https://github.com/spf13/viper/issues/871
func (v *Viper) SubExact(key string) (*Viper, error) {
	data := v.Get(key)
	if data == nil {
		return NewViper(), nil
	}

	if reflect.TypeOf(data).Kind() == reflect.Map {
		subv := NewViper()
		// Cannot return error because the subv is empty.
		_ = subv.MergeConfigMap(cast.ToStringMap(data))
		return subv, nil
	}
	return nil, fmt.Errorf("unexpected sub-config value kind for key:%s value:%v kind:%v)", key, data, reflect.TypeOf(data).Kind())
}

// Creates a new Viper instance with a different key-delimitor "::" instead of the
// default ".". This way configs can have keys that contain ".".
func NewViper() *Viper {
	return &Viper{Viper: viper.NewWithOptions(viper.KeyDelimiter(ViperDelimiter))}
}
