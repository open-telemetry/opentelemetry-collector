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

// package configload implements the configuration Loader struct.
package configload

import (
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

// NewLoader creates a new Loader instance.
func NewLoader() *Loader {
	return &Loader{
		v: NewViper(),
	}
}

// FromViper creates a Loader from a Viper instance.
func FromViper(v *viper.Viper) *Loader {
	return &Loader{v: v}
}

// Loader loads configuration.
type Loader struct {
	v *viper.Viper
}

// Viper returns the underlying Viper instance.
func (l *Loader) Viper() *viper.Viper {
	return l.v
}

// UnmarshalExact unmarshals the config into a struct, erroring if a field is nonexistent.
func (l *Loader) UnmarshalExact(intoCfg interface{}) error {
	return l.v.UnmarshalExact(intoCfg)
}

// ToStringMap creates a map[string]interface{} from a ConfigLoader.
func (l *Loader) ToStringMap() map[string]interface{} {
	return cast.ToStringMap(l.v)
}
